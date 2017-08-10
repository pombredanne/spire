package devices

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/superscale/spire/config"
	"github.com/superscale/spire/mqtt"
)

// MessageHandler ...
type MessageHandler struct {
	broker      *mqtt.Broker
	formations  *FormationMap
	idleTimeout time.Duration
}

// NewMessageHandler ...
func NewMessageHandler(broker *mqtt.Broker) *MessageHandler {
	return &MessageHandler{
		broker:      broker,
		formations:  NewFormationMap(),
		idleTimeout: config.Config.IdleConnectionTimeout,
	}
}

// HandleConnection receives a connection from a device and dispatches its messages to the designated handler
func (h *MessageHandler) HandleConnection(conn *mqtt.Conn) {
	connectPkg, err := conn.ReadConnect()
	if err != nil {
		log.Println("error while reading packet:", err, ". closing connection")
		conn.Close()
		return
	}

	deviceName := connectPkg.ClientIdentifier
	formationID := connectPkg.Username

	if len(formationID) == 0 {
		log.Println("CONNECT packet from", conn.RemoteAddr(), "is missing formation ID. closing connection")
		conn.Close()
		return
	}

	deviceDisconnected, err := h.deviceConnected(formationID, deviceName, conn)
	if err != nil {
		log.Println(err)
		conn.Close()
		return
	}

	conn.AcknowledgeConnect()
	h.handleMessages(conn, formationID, deviceName, deviceDisconnected)
}

func (h *MessageHandler) handleMessages(conn *mqtt.Conn, formationID, deviceName string, deviceDisconnected func()) {
	for {
		ca, err := conn.Read()
		if err != nil {
			if err != io.EOF {
				log.Printf("error while reading packet from %s: %v. closing connection", deviceName, err)
			}

			deviceDisconnected()
			return
		}

		switch ca := ca.(type) {
		case *packets.PingreqPacket:
			err = conn.SendPong()
		case *packets.PublishPacket:
			err = h.dispatch(formationID, deviceName, ca)
		case *packets.SubscribePacket:
			err = h.broker.Subscribe(ca, conn)
		case *packets.UnsubscribePacket:
			err = h.broker.Unsubscribe(ca, conn)
		case *packets.DisconnectPacket:
			deviceDisconnected()
			return
		default:
			log.Println("ignoring unsupported message from", deviceName)
		}

		if err != nil {
			log.Printf("error while handling packet from device %s (%v): %v", deviceName, conn.RemoteAddr(), err)
		}
	}
}

// GetDeviceState only exists to observe state changes in tests :(
func (h *MessageHandler) GetDeviceState(formationID, deviceName, key string) interface{} {
	return h.formations.GetDeviceState(formationID, deviceName, key)
}

func (h *MessageHandler) deviceConnected(formationID, deviceName string, conn *mqtt.Conn) (func(), error) {
	info, err := fetchDeviceInfo(deviceName)
	if err != nil {
		return nil, err
	}

	deviceOS := getDeviceOS(info)
	h.formations.PutDeviceState(formationID, deviceName, "device_info", map[string]interface{}{"device_os": deviceOS})

	if err := h.initOTAState(formationID, deviceName); err != nil {
		log.Println("failed to init OTA state.", err)
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	go h.publishUpState(ctx, deviceName)

	disconnectFn := func() {
		cancelFn()
		h.broker.Remove(conn)

		if err := conn.Close(); err != nil {
			log.Println(err)
		}

		// FIXME this does not belong here
		otaState, ok := h.formations.GetDeviceState(formationID, deviceName, "ota").(*OTAState)
		if ok && otaState.State == Downloading {

			otaState := &OTAState{
				State: Error,
				Error: "connection to device lost during download",
			}

			publishOTAMessage(deviceName, otaState, h.broker)
		}
	}

	return disconnectFn, nil
}

func (h *MessageHandler) initOTAState(formationID, deviceName string) error {
	state := &OTAState{State: Default}
	h.formations.PutDeviceState(formationID, deviceName, "ota", state)
	return publishOTAMessage(deviceName, state, h.broker)
}

func (h *MessageHandler) publishUpState(ctx context.Context, deviceName string) {
	upState := map[string]interface{}{
		"state":     "up",
		"timestamp": time.Now().Unix(),
	}

	send := func(state string) {
		upState["state"] = state
		pkg, err := mqtt.MakePublishPacket("/armada/"+deviceName+"/up", upState)

		if err != nil {
			log.Println(err)
		} else {
			h.broker.Publish(pkg)
		}
	}
	send("up")

	for {
		select {
		case <-ctx.Done():
			send("down")
			return
		case <-time.After(30 * time.Second):
			send("up")
		}
	}
}

func (h *MessageHandler) dispatch(formationID, deviceName string, msg *packets.PublishPacket) (err error) {
	parts := strings.Split(msg.TopicName, "/")
	if len(parts) < 4 || parts[0] != "" || parts[1] != "pylon" || parts[2] != deviceName {
		return
	}

	switch strings.Join(parts[3:], "/") {
	case "wan/ping":
		err = HandlePing(msg.TopicName, msg.Payload, formationID, deviceName, h.formations, h.broker)
	case "exceptions":
		err = HandleException(msg.TopicName, msg.Payload, formationID, deviceName, h.formations, h.broker)
	case "ota":
		err = HandleOTA(msg.TopicName, msg.Payload, formationID, deviceName, h.formations, h.broker)
	default:
		break
	}
	return
}

func fetchDeviceInfo(deviceName string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v2/devices/%s", config.Config.LiberatorBaseURL, deviceName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+config.Config.LiberatorJWTToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	info := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		err := fmt.Errorf("unexpected response from liberator for device %s. status: %d %s. error: %v",
			deviceName, resp.StatusCode, resp.Status, info["error"])

		return nil, err
	}

	return info, nil
}

func getDeviceOS(info map[string]interface{}) (res string) {
	res = "unknown"

	data, ok := info["data"].(map[string]interface{})
	if ok {
		sysimg, ok := data["current_system_image"].(map[string]interface{})
		if ok {
			vendor, ok := sysimg["vendor"].(string)
			if !ok {
				return
			}
			product, ok := sysimg["product"].(string)
			if !ok {
				return
			}
			variant, ok := sysimg["variant"].(string)
			if !ok {
				return
			}
			version, ok := sysimg["version"].(float64)
			if !ok {
				return
			}
			res = fmt.Sprintf("%s-%s-%s-%d", vendor, product, variant, int(version))
		}
	}

	return
}

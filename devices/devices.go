package devices

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/superscale/spire/mqtt"
	"github.com/superscale/spire/service"
)

// MessageHandler ...
type MessageHandler struct {
	broker     *mqtt.Broker
	formations *FormationMap
}

// NewMessageHandler ...
func NewMessageHandler(broker *mqtt.Broker) *MessageHandler {
	return &MessageHandler{
		broker:     broker,
		formations: NewFormationMap(),
	}
}

// HandleConnection receives a connection from a device and dispatches its messages to the designated handler
func (h *MessageHandler) HandleConnection(conn net.Conn) {
	connectPkg, err := mqtt.Connect(conn, false)
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

	cAck := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	err = cAck.Write(conn)
	h.handleMessages(conn, formationID, deviceName, deviceDisconnected)
}

func (h *MessageHandler) handleMessages(conn net.Conn, formationID, deviceName string, deviceDisconnected func()) {
	for {
		ca, err := packets.ReadPacket(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("error while reading packet from %s: %v. closing connection", deviceName, err)
			}

			deviceDisconnected()
			return
		}

		switch ca := ca.(type) {
		case *packets.PingreqPacket:
			if err := mqtt.SendPingResponse(conn); err != nil {
				return
			}
		case *packets.PublishPacket:
			h.dispatch(formationID, deviceName, ca)
		case *packets.SubscribePacket:
			h.broker.Subscribe(ca, conn)
		case *packets.UnsubscribePacket:
			h.broker.Unsubscribe(ca, conn)
		case *packets.DisconnectPacket:
			deviceDisconnected()
			return
		default:
			log.Println("ignoring unsupported message from", deviceName)
		}
	}
}

// GetDeviceState only exists to observe state changes in tests :(
func (h *MessageHandler) GetDeviceState(formationID, deviceName, key string) interface{} {
	return h.formations.GetDeviceState(formationID, deviceName, key)
}

func (h *MessageHandler) deviceConnected(formationID, deviceName string, conn net.Conn) (func(), error) {
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

func (h *MessageHandler) dispatch(formationID, deviceName string, msg *packets.PublishPacket) {
	parts := strings.Split(msg.TopicName, "/")
	if len(parts) < 4 || parts[0] != "" || parts[1] != "pylon" || parts[2] != deviceName {
		return
	}

	switch strings.Join(parts[3:], "/") {
	case "wan/ping":
		if err := HandlePing(msg.TopicName, msg.Payload, formationID, deviceName, h.formations, h.broker); err != nil {
			log.Println(err)
		}
	case "exceptions":
		if err := HandleException(msg.TopicName, msg.Payload, formationID, deviceName, h.formations, h.broker); err != nil {
			log.Println(err)
		}
	case "ota":
		if err := HandleOTA(msg.TopicName, msg.Payload, formationID, deviceName, h.formations, h.broker); err != nil {
			log.Println(err)
		}
	default:
		return
	}
}

func fetchDeviceInfo(deviceName string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v2/devices/%s", service.Config.LiberatorBaseURL, deviceName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+service.Config.LiberatorJWTToken)
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

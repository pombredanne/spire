package devices

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/superscale/spire/config"
	"github.com/superscale/spire/mqtt"
)

// ConnectTopic ...
const ConnectTopic = "/spire/devices/connect"

// DisconnectTopic ...
const DisconnectTopic = "/spire/devices/disconnect"

// ConnectMessage ...
type ConnectMessage struct {
	FormationID string
	DeviceName  string
	DeviceInfo  map[string]interface{}
}

// DisconnectMessage ...
type DisconnectMessage struct {
	FormationID string
	DeviceName  string
}

// Handler ...
type Handler struct {
	broker     *mqtt.Broker
}

// NewHandler ...
func NewHandler(broker *mqtt.Broker) *Handler {
	return &Handler{
		broker:     broker,
	}
}

// Topic ...
type Topic struct {
	Prefix     string
	DeviceName string
	Path       string
}

// ParseTopic ...
func ParseTopic(topic string) Topic {
	parts := strings.SplitN(topic, "/", 4)
	return Topic{parts[1], parts[2], "/" + parts[3]}
}

// HandleConnection ...
func (h *Handler) HandleConnection(session *mqtt.Session) {
	connectPkg, err := session.ReadConnect()
	if err != nil {
		log.Println("error while reading packet:", err, ". closing connection")
		session.Close()
		return
	}

	deviceName := connectPkg.ClientIdentifier
	formationID := connectPkg.Username

	if len(formationID) == 0 {
		log.Println("CONNECT packet from", session.RemoteAddr(), "is missing formation ID. closing connection")
		session.Close()
		return
	}

	if err := h.deviceConnected(formationID, deviceName, session); err != nil {
		log.Println(err)
		session.Close()
		return
	}

	session.AcknowledgeConnect()
	h.handlePackets(formationID, deviceName, session)
}

func (h *Handler) handlePackets(formationID, deviceName string, session *mqtt.Session) {
	for {
		ca, err := session.Read()
		if err != nil {
			if err != io.EOF {
				log.Printf("error while reading packet from %s: %v. closing connection", deviceName, err)
			}

			h.deviceDisconnected(formationID, deviceName, session)
			return
		}

		switch ca := ca.(type) {
		case *packets.PingreqPacket:
			err = session.SendPingresp()
		case *packets.PublishPacket:
			h.broker.Publish(ca.TopicName, ca.Payload)
		case *packets.SubscribePacket:
			h.broker.SubscribeAll(ca, session.Publish)
			err = session.SendSuback(ca.MessageID)
		case *packets.UnsubscribePacket:
			h.broker.UnsubscribeAll(ca, session.Publish)
			err = session.SendUnsuback(ca.MessageID)
		case *packets.DisconnectPacket:
			h.deviceDisconnected(formationID, deviceName, session)
			return
		default:
			log.Println("ignoring unsupported message from", deviceName)
		}

		if err != nil {
			log.Printf("error while handling packet from device %s (%v): %v", deviceName, session.RemoteAddr(), err)
		}
	}
}

func (h *Handler) deviceConnected(formationID, deviceName string, session *mqtt.Session) error {
	info, err := fetchDeviceInfo(deviceName)
	if err != nil {
		return err
	}

	h.broker.Publish(ConnectTopic, &ConnectMessage{formationID, deviceName, info})
	return nil
}

func (h *Handler) deviceDisconnected(formationID, deviceName string, session *mqtt.Session) {
	h.broker.Remove(session.Publish)

	if err := session.Close(); err != nil {
		log.Println(err)
	}

	h.broker.Publish(DisconnectTopic, &DisconnectMessage{formationID, deviceName})
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

package devices

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
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
	FormationID string `json:"formation_id"`
	DeviceName  string
	DeviceInfo  map[string]interface{}
	IPAddress   string `json:"ip_address"`
}

// DisconnectMessage ...
type DisconnectMessage struct {
	FormationID string
	DeviceName  string
}

// Handler ...
type Handler struct {
	broker *mqtt.Broker
}

// NewHandler ...
func NewHandler(broker *mqtt.Broker) *Handler {
	return &Handler{
		broker: broker,
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

	cm, err := h.connect(session)
	if err != nil {
		if err != io.EOF {
			log.Println(err)
			session.Close()
		}
		return
	}

	for {
		ca, err := session.Read()
		if err != nil {
			if err != io.EOF {
				log.Printf("error while reading packet from %s: %v. closing connection", cm.DeviceName, err)
			}

			h.deviceDisconnected(cm.FormationID, cm.DeviceName, session)
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
			h.deviceDisconnected(cm.FormationID, cm.DeviceName, session)
			return
		default:
			log.Println("ignoring unsupported message from", cm.DeviceName)
		}

		if err != nil {
			log.Printf("error while handling packet from device %s (%v): %v", cm.DeviceName, session.RemoteAddr(), err)
		}
	}
}

func (h *Handler) connect(session *mqtt.Session) (*ConnectMessage, error) {
	pkg, err := session.ReadConnect()
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, fmt.Errorf("error while reading packet: %v. closing connection", err)
	}

	cm := &ConnectMessage{DeviceName: pkg.ClientIdentifier}
	if err := json.Unmarshal([]byte(pkg.Username), cm); err != nil {
		return nil, err
	}

	if len(cm.FormationID) == 0 {
		return nil, fmt.Errorf("CONNECT packet from %v is missing formation ID. closing connection", session.RemoteAddr())
	}

	cm.DeviceInfo, err = fetchDeviceInfo(cm.DeviceName)
	if err != nil {
		return nil, err
	}

	if err = session.AcknowledgeConnect(); err != nil {
		return nil, err
	}

	h.broker.Publish(ConnectTopic, cm)
	return cm, nil
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

// Round ...
func Round(f, places float64) float64 {
	shift := math.Pow(10, places)
	f = math.Floor(f*shift + .5)
	return f / shift
}

package devices

import (
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/superscale/spire/mqtt"
)

// State is a crappy name
type State struct {
	FormationState *syncMap
	Devices        *DeviceMap
}

// NewState ...
func NewState() *State {
	return &State{
		FormationState: newSyncMap(),
		Devices:        newDeviceMap(),
	}
}

// MessageHandler ...
type MessageHandler struct {
	state  *State
	broker *mqtt.Broker
}

// NewMessageHandler ...
func NewMessageHandler(state *State, broker *mqtt.Broker) *MessageHandler {
	return &MessageHandler{
		state:  state,
		broker: broker,
	}
}

// HandleConnection receives a connection from a device and dispatches its messages to the designated handler
func (h *MessageHandler) HandleConnection(conn net.Conn) {
	connectPkg, err := mqtt.Connect(conn)
	if err != nil {
		log.Println("error while reading packet:", err, "closing connection")
		conn.Close()
		return
	}

	deviceName := connectPkg.ClientIdentifier
	if err := h.DeviceConnected(deviceName, conn); err != nil {
		log.Println(err)
	}

	for {
		ca, err := packets.ReadPacket(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("error while reading packet from %s: %v. closing connection", deviceName, err)
			}

			if err := h.DeviceDisconnected(deviceName, conn); err != nil {
				log.Println(err)
			}
			return
		}

		switch ca := ca.(type) {
		case *packets.PublishPacket:
			if err := h.dispatch(deviceName, ca); err != nil {
				log.Println(err)
			}

			h.broker.Publish(ca)
		case *packets.SubscribePacket:
			h.broker.Subscribe(ca, conn)
		case *packets.UnsubscribePacket:
			h.broker.Unsubscribe(ca, conn)
		case *packets.DisconnectPacket:
			if err := h.DeviceDisconnected(deviceName, conn); err != nil {
				log.Println(err)
			}
			return
		default:
			log.Println("ignoring unsupported message from", deviceName)
		}
	}
}

// DeviceConnected ...
func (h *MessageHandler) DeviceConnected(deviceName string, conn net.Conn) (err error) {
	dev := h.state.Devices.Add(deviceName, conn)
	dev.PutState("up", map[string]interface{}{"state": "up", "timestamp": time.Now().Unix()})
	return
}

// DeviceDisconnected ...
func (h *MessageHandler) DeviceDisconnected(deviceName string, conn net.Conn) (err error) {
	h.broker.Remove(conn)

	if err := conn.Close(); err != nil {
		log.Println(err)
	}

	dev, err := h.state.Devices.Get(deviceName)
	if err != nil {
		return
	}

	dev.PutState("up", map[string]interface{}{"state": "down", "timestamp": time.Now().Unix()})
	return
}

func (h *MessageHandler) dispatch(deviceName string, msg *packets.PublishPacket) error {
	parts := strings.Split(msg.TopicName, "/")
	if len(parts) < 4 || parts[0] != "" || parts[1] != "pylon" || parts[2] != deviceName {
		return nil
	}

	switch parts[3] {
	case "ping":
		return HandlePing(deviceName, msg.TopicName, msg.Payload, h.state, h.broker)
	default:
		return nil
	}
}

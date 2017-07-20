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

// MessageHandler ...
type MessageHandler struct {
	broker     *mqtt.Broker
	formations *formationMap
}

// NewMessageHandler ...
func NewMessageHandler(broker *mqtt.Broker) *MessageHandler {
	return &MessageHandler{
		broker:     broker,
		formations: newFormationMap(),
	}
}

// HandleConnection receives a connection from a device and dispatches its messages to the designated handler
func (h *MessageHandler) HandleConnection(conn net.Conn) {
	connectPkg, err := mqtt.Connect(conn)
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

	h.deviceConnected(formationID, deviceName, conn)

	for {
		ca, err := packets.ReadPacket(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("error while reading packet from %s: %v. closing connection", deviceName, err)
			}

			h.deviceDisconnected(formationID, deviceName, conn)
			return
		}

		switch ca := ca.(type) {
		case *packets.PublishPacket:
			h.dispatch(formationID, deviceName, ca)
			h.broker.Publish(ca)
		case *packets.SubscribePacket:
			h.broker.Subscribe(ca, conn)
		case *packets.UnsubscribePacket:
			h.broker.Unsubscribe(ca, conn)
		case *packets.DisconnectPacket:
			h.deviceDisconnected(formationID, deviceName, conn)
			return
		default:
			log.Println("ignoring unsupported message from", deviceName)
		}
	}
}

// GetDeviceState only exists to observe state changes in tests :(
func (h *MessageHandler) GetDeviceState(formationID, deviceName, key string) interface{} {
	return h.formations.getDeviceState(formationID, deviceName, key)
}

func (h *MessageHandler) deviceConnected(formationID, deviceName string, conn net.Conn) {
	h.formations.putDeviceState(formationID, deviceName, "up", map[string]interface{}{"state": "up", "timestamp": time.Now().Unix()})
}

func (h *MessageHandler) deviceDisconnected(formationID, deviceName string, conn net.Conn) {
	h.broker.Remove(conn)

	if err := conn.Close(); err != nil {
		log.Println(err)
	}

	h.formations.putDeviceState(formationID, deviceName, "up", map[string]interface{}{"state": "down", "timestamp": time.Now().Unix()})
}

func (h *MessageHandler) dispatch(formationID, deviceName string, msg *packets.PublishPacket) {
	parts := strings.Split(msg.TopicName, "/")
	if len(parts) < 4 || parts[0] != "" || parts[1] != "pylon" || parts[2] != deviceName {
		return
	}

	formation := h.formations.get(formationID)
	// should never happen since the formation state is initialized when a device connects
	if formation == nil {
		log.Println("no formation state found for formationID", formationID)
		return
	}

	switch parts[3] {
	case "ping":
		if err := HandlePing(msg.TopicName, msg.Payload, formation, h.broker); err != nil {
			log.Println(err)
		}
	default:
		return
	}
}

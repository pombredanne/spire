package control

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
	"github.com/superscale/spire/service"
)

// MessageHandler ...
type MessageHandler struct {
	state  *devices.State
	broker *service.Broker
}

// NewMessageHandler ...
func NewMessageHandler(state *devices.State, broker *service.Broker) *MessageHandler {
	return &MessageHandler{
		state:  state,
		broker: broker,
	}
}

// HandleConnection ...
func (h *MessageHandler) HandleConnection(conn net.Conn) {
	if _, err := mqtt.Connect(conn); err != nil {
		log.Println(err)
		return
	}

	for {
		ca, err := packets.ReadPacket(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("error while reading packet: %v. closing connection", err)
				conn.Close()
			}
			return
		}

		switch ca := ca.(type) {
		case *packets.DisconnectPacket:
			h.broker.Remove(conn)
			conn.Close()
		case *packets.PublishPacket:
			if err := forwardPacket(ca, h.state); err != nil {
				log.Println(err)
			}
		case *packets.SubscribePacket:
			h.broker.Subscribe(ca, conn)
		default:
			conn.Close()
			return
		}
	}
}

func forwardPacket(pkg *packets.PublishPacket, state *devices.State) error {
	if pkg.Qos > 0 {
		panic("QoS > 0 not supported")
	}

	parts := strings.Split(pkg.TopicName, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid topic: %s", pkg.TopicName)
	}

	// currently liberator sends control messages to "/armada/1.marsara/foo" but devices subscribe to "/pylon/1.marsara/foo"
	// we should fix this so we can use pub/sub in the broker and remove this special case
	parts[1] = "pylon"
	return state.Devices.Send(parts[2], strings.Join(parts, "/"), pkg.Payload)
}

package control

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/superscale/spire/devices"
)

// MessageHandler ...
type MessageHandler struct {
	devices *devices.DeviceMap
}

// NewMessageHandler ...
func NewMessageHandler(devices *devices.DeviceMap) *MessageHandler {
	return &MessageHandler{
		devices: devices,
	}
}

// HandleConnection ...
func (h *MessageHandler) HandleConnection(conn net.Conn) {
	if err := handshake(conn); err != nil {
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
			conn.Close()
			return
		case *packets.PublishPacket:
			if err := forwardPacket(ca, h.devices); err != nil {
				log.Println(err)
			} else {
				pAck := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
				pAck.MessageID = ca.MessageID
				pAck.Write(conn)
			}
		case *packets.SubscribePacket:
			sAck := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
			sAck.MessageID = ca.MessageID
			sAck.Write(conn)
		default:
			log.Println("ignoring unsupported message")
		}
	}
}

func forwardPacket(pkg *packets.PublishPacket, devs *devices.DeviceMap) error {
	parts := strings.Split(pkg.TopicName, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid topic: %s", pkg.TopicName)
	}

	// currently liberator sends control messages to "/armada/1.marsara/foo" but devices subscribe to "/pylon/1.marsara/foo"
	parts[1] = "pylon"
	return devs.Send(parts[2], strings.Join(parts, "/"), pkg.Payload)
}

func handshake(conn net.Conn) (err error) {
	ca, err := packets.ReadPacket(conn)
	if err != nil {
		return
	}
	if _, ok := ca.(*packets.ConnectPacket); !ok {
		return errors.New("expected a CONNECT message, got some other garbage instead. closing connection")
	}

	cAck := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	return cAck.Write(conn)
}

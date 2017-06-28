package handlers

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/superscale/spire/devices"
)

type DeviceMessageHandler struct {
	devices *devices.DeviceMap
}

func NewDeviceMessageHandler(devices *devices.DeviceMap) *DeviceMessageHandler {
	return &DeviceMessageHandler{
		devices: devices,
	}
}

// HandleConnection receives a connection from a device and dispatches its messages to the designated handler
func (h *DeviceMessageHandler) HandleConnection(conn net.Conn) {
	ca, err := packets.ReadPacket(conn)
	if err != nil {
		log.Println("error while reading packet:", err, "closing connection")
		conn.Close()
		return
	}

	msg, ok := ca.(*packets.ConnectPacket)
	if !ok {
		log.Println("expected a CONNECT message, got some other garbage instead. closing connection")
		conn.Close()
		return
	}

	cAck := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	if err := cAck.Write(conn); err != nil {
		log.Println(err)
		conn.Close()
		return
	}
	deviceName := msg.ClientIdentifier
	h.devices.Add(deviceName, conn)

	for {
		ca, err := packets.ReadPacket(conn)
		if err != nil {
			log.Printf("error while reading packet from %s: %v. closing connection", deviceName, err)
			if err := conn.Close(); err != nil {
				log.Println(err)
			}
			return
		}

		switch ca := ca.(type) {
		case *packets.DisconnectPacket:
			log.Println(deviceName, "signing off. closing connection")
			if err := conn.Close(); err != nil {
				log.Println(err)
			}
			return
		case *packets.PublishPacket:
			if err := dispatch(deviceName, ca, h.devices); err != nil {
				log.Println(err)
			}

			pAck := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
			pAck.MessageID = ca.MessageID
			pAck.Write(conn)
		default:
			log.Println("ignoring unsupported message from", deviceName)
		}
	}
}

func dispatch(deviceName string, msg *packets.PublishPacket, devs *devices.DeviceMap) error {
	parts := strings.Split(msg.TopicName, "/")
	if parts[0] != "" || parts[1] != "pylon" || parts[2] != deviceName || len(parts) < 4 {
		return fmt.Errorf("invalid message received from %s topic: %s payload: %s", deviceName, msg.TopicName, string(msg.Payload))
	}

	switch parts[3] {
	case "ping":
		return HandlePing(deviceName, msg.TopicName, msg.Payload, devs)
	default:
		return fmt.Errorf("unsupported message received from %s topic: %s payload: %s", deviceName, msg.TopicName, string(msg.Payload))
	}

	return nil
}

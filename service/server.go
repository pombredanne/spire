package service

import (
	"log"
	"net"

	"fmt"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/superscale/spire/handlers"
	"strings"
)

// Server ...
type Server struct {
	listener net.Listener
}

// Run ...
func (s *Server) Run() (err error) {
	if s.listener, err = net.Listen("tcp", Config.Bind); err != nil {
		return
	}

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Println(err)
		} else {
			go handleConnection(conn)
		}
	}
}

func handleConnection(conn net.Conn) {
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

	handleMessages(conn, msg.ClientIdentifier)
}

func handleMessages(conn net.Conn, deviceName string) {
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
			if err := dispatch(conn, deviceName, ca); err != nil {
				log.Println(err)
			}
		default:
			log.Println("ignoring unsupported message from", deviceName)
		}
	}
}

func dispatch(conn net.Conn, deviceName string, msg *packets.PublishPacket) error {
	if msg.Qos != 1 {
		return fmt.Errorf("received publish message with unsupported QoS %d from %s", msg.Qos, deviceName)
	}

	parts := strings.Split(msg.TopicName, "/")
	if parts[0] != "" || parts[1] != "pylon" || parts[2] != deviceName || len(parts) < 4 {
		return fmt.Errorf("invalid message received from %s topic: %s payload: %s", deviceName, msg.TopicName, string(msg.Payload))

	}

	switch parts[3] {
	case "ping":
		handlers.HandlePing(deviceName, msg.TopicName, msg.Payload)
	default:
		return fmt.Errorf("unsupported message received from %s topic: %s payload: %s", deviceName, msg.TopicName, string(msg.Payload))
	}

	pAck := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
	pAck.MessageID = msg.MessageID
	pAck.Write(conn)
	return nil
}

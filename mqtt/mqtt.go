package mqtt

import (
	"encoding/json"
	"errors"
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// Connect performs the connection handshake and returns the connect packet or an error
func Connect(conn net.Conn, sendConnack bool) (p *packets.ConnectPacket, err error) {
	ca, err := packets.ReadPacket(conn)
	if err != nil {
		return
	}

	var ok bool
	if p, ok = ca.(*packets.ConnectPacket); !ok {
		return nil, errors.New("expected a CONNECT message, got some other garbage instead. closing connection")
	}

	if sendConnack {
		cAck := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
		err = cAck.Write(conn)
	}
	return
}

// MakePublishPacket creates a packet with QoS 0 and does JSON marshalling unless payload is of type []byte
func MakePublishPacket(topic string, message interface{}) (p *packets.PublishPacket, err error) {
	var payload []byte

	switch m := message.(type) {
	case []byte:
		payload = m
	default:
		payload, err = json.Marshal(m)
		if err != nil {
			return
		}
	}

	p = packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
	p.Qos = 0
	p.TopicName = topic
	p.Payload = payload
	return
}

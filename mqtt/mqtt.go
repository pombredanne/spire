package mqtt

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// Conn represents an MQTT connection
// TODO Either implement the net.Conn interface, or change the name.
type Conn struct {
	conn        net.Conn
	idleTimeout time.Duration
}

// NewConn returns a new mqtt.Conn
func NewConn(conn net.Conn, idleTimeout time.Duration) *Conn {
	return &Conn{
		conn:        conn,
		idleTimeout: idleTimeout,
	}
}

// ReadConnect reads the connect packet or times out
func (c *Conn) ReadConnect() (p *packets.ConnectPacket, err error) {
	c.conn.SetReadDeadline(c.deadline())

	var ca packets.ControlPacket
	if ca, err = packets.ReadPacket(c.conn); err != nil {
		return
	}

	var ok bool
	if p, ok = ca.(*packets.ConnectPacket); !ok {
		return nil, fmt.Errorf("expected a CONNECT packet from %v, got this instead: %s", c.conn.RemoteAddr(), ca.String())
	}

	return
}

// AcknowledgeConnect ...
func (c *Conn) AcknowledgeConnect() error {
	cAck := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	c.conn.SetWriteDeadline(c.deadline())
	return cAck.Write(c.conn)
}

// Handshake performs the connection handshake and returns the connect packet or an error
func (c *Conn) Handshake() (p *packets.ConnectPacket, err error) {
	if p, err = c.ReadConnect(); err != nil {
		return
	}

	if err = c.AcknowledgeConnect(); err != nil {
		p = nil
	}
	return
}

// Close ...
func (c *Conn) Close() error {
	return c.conn.Close()
}

// RemoteAddr ...
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Read a packet or time out
func (c *Conn) Read() (p packets.ControlPacket, err error) {
		c.conn.SetReadDeadline(c.deadline())
		return packets.ReadPacket(c.conn)
}

// Write a packet or time out
func (c *Conn) Write(pkg packets.ControlPacket) error {
	c.conn.SetWriteDeadline(c.deadline())
	return pkg.Write(c.conn)
}

// SendPong ...
func (c *Conn) SendPong() error {
	resp := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	return c.Write(resp)
}

func (c *Conn) deadline() time.Time {
	return time.Now().Add(c.idleTimeout)
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

// Pipe ...
func Pipe() (*Conn, *Conn) {
	a, b := net.Pipe()
	t := time.Second * 1
	return NewConn(a, t), NewConn(b, t)
}

package devices

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// MessageHandler ...
type MessageHandler struct {
	devices *DeviceMap
}

// NewMessageHandler ...
func NewMessageHandler(devices *DeviceMap) *MessageHandler {
	return &MessageHandler{
		devices: devices,
	}
}

// HandleConnection receives a connection from a device and dispatches its messages to the designated handler
func (h *MessageHandler) HandleConnection(conn net.Conn) {
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

	deviceName := msg.ClientIdentifier
	h.devices.Add(deviceName, conn)

	cAck := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	if err := cAck.Write(conn); err != nil {
		log.Println(err)
		conn.Close()
		h.devices.Remove(deviceName)
		return
	}

	for {
		ca, err := packets.ReadPacket(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("error while reading packet from %s: %v. closing connection", deviceName, err)
			}
			if err := conn.Close(); err != nil {
				log.Println(err)
			}

			h.devices.Remove(deviceName)
			return
		}

		switch ca := ca.(type) {
		case *packets.DisconnectPacket:
			if err := conn.Close(); err != nil {
				log.Println(err)
			}
			h.devices.Remove(deviceName)
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

func dispatch(deviceName string, msg *packets.PublishPacket, devs *DeviceMap) error {
	parts := strings.Split(msg.TopicName, "/")
	if len(parts) < 4 || parts[0] != "" || parts[1] != "pylon" || parts[2] != deviceName {
		return fmt.Errorf("invalid message received from %s topic: %s payload: %s", deviceName, msg.TopicName, string(msg.Payload))
	}

	switch parts[3] {
	case "ping":
		return HandlePing(deviceName, msg.TopicName, msg.Payload, devs)
	default:
		return fmt.Errorf("unsupported message received from %s topic: %s payload: %s", deviceName, msg.TopicName, string(msg.Payload))
	}
}

// Device ...
type Device struct {
	name string
	conn net.Conn
}

// NewDevice ...
func NewDevice(name string, conn net.Conn) *Device {
	return &Device{
		name: name,
		conn: conn,
	}
}

// Send ...
func (d *Device) Send(topic string, payload []byte) error {
	pkg := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
	pkg.Qos = 1
	pkg.TopicName = topic
	pkg.Payload = payload
	return pkg.Write(d.conn)
}

// ErrDevNotFound ...
var ErrDevNotFound = errors.New("device not found")

// DeviceMap ...
type DeviceMap struct {
	l sync.RWMutex
	m map[string]*Device
}

// NewDeviceMap ...
func NewDeviceMap() *DeviceMap {
	return &DeviceMap{
		m: make(map[string]*Device),
	}
}

// Get ...
func (d *DeviceMap) Get(name string) (*Device, error) {
	d.l.RLock()
	defer d.l.RUnlock()

	dev, exists := d.m[name]
	if !exists {
		return nil, ErrDevNotFound
	}
	return dev, nil
}

// Add ...
func (d *DeviceMap) Add(name string, conn net.Conn) {
	d.l.Lock()
	defer d.l.Unlock()

	dev, exists := d.m[name]
	if exists {
		if dev.conn == conn {
			return
		}

		dev.conn.Close()
	}
	d.m[name] = NewDevice(name, conn)
}

// Remove ...
func (d *DeviceMap) Remove(name string) {
	d.l.Lock()
	defer d.l.Unlock()

	dev, exists := d.m[name]
	if !exists {
		return
	}

	dev.conn.Close()
	delete(d.m, name)
}

// Send ...
func (d *DeviceMap) Send(name, topic string, payload []byte) error {
	d.l.RLock()
	defer d.l.RUnlock()

	dev, exists := d.m[name]
	if !exists {
		return ErrDevNotFound
	}

	return dev.Send(topic, payload)
}

// Broadcast ...
func (d *DeviceMap) Broadcast(topic string, payload []byte) error {
	d.l.RLock()
	defer d.l.RUnlock()

	for _, dev := range d.m {
		if err := dev.Send(topic, payload); err != nil {
			return err
		}
	}
	return nil
}

package devices

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

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
	if err := h.DeviceConnected(deviceName, conn); err != nil {
		log.Println(err)
	}

	cAck := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	if err := cAck.Write(conn); err != nil {
		log.Println(err)
		conn.Close()
		if err := h.DeviceDisconnected(deviceName); err != nil {
			log.Println(err)
		}
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

			if err := h.DeviceDisconnected(deviceName); err != nil {
				log.Println(err)
			}
			return
		}

		switch ca := ca.(type) {
		case *packets.DisconnectPacket:
			if err := conn.Close(); err != nil {
				log.Println(err)
			}
			if err := h.DeviceDisconnected(deviceName); err != nil {
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

// DeviceConnected ...
func (h *MessageHandler) DeviceConnected(deviceName string, conn net.Conn) (err error) {
	dev := h.devices.Add(deviceName, conn)
	dev.State.Put("up", map[string]interface{}{"state": "up", "timestamp": time.Now().Unix()})
	return
}

// DeviceDisconnected ...
func (h *MessageHandler) DeviceDisconnected(deviceName string) (err error) {
	dev, err := h.devices.Get(deviceName)
	if err != nil {
		return
	}

	dev.State.Put("up", map[string]interface{}{"state": "down", "timestamp": time.Now().Unix()})
	return
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

// DeviceState ...
type DeviceState struct {
	l sync.RWMutex
	m map[string]interface{}
}

// NewDeviceState ...
func NewDeviceState() *DeviceState {
	return &DeviceState{
		m: make(map[string]interface{}),
	}
}

// Get ...
func (s *DeviceState) Get(key string) (interface{}, bool) {
	s.l.RLock()
	defer s.l.RUnlock()

	value, exists := s.m[key]
	return value, exists
}

// Put ...
func (s *DeviceState) Put(key string, value interface{}) {
	s.l.Lock()
	defer s.l.Unlock()

	s.m[key] = value
}

// Device ...
type Device struct {
	State *DeviceState
	name  string
	conn  net.Conn
}

// NewDevice ...
func NewDevice(name string, conn net.Conn) *Device {
	return &Device{
		State: NewDeviceState(),
		name:  name,
		conn:  conn,
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
func (d *DeviceMap) Add(name string, conn net.Conn) *Device {
	d.l.Lock()
	defer d.l.Unlock()

	dev, exists := d.m[name]
	if exists {
		if dev.conn != conn {
			dev.conn = conn
		}

		return dev
	}

	dev = NewDevice(name, conn)
	d.m[name] = dev
	return dev
}

// Send ...
func (d *DeviceMap) Send(name, topic string, payload []byte) error {
	dev, err := d.Get(name)
	if err != nil {
		return err
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

package devices

import (
	"errors"
	"net"
	"sync"

	"github.com/superscale/spire/mqtt"
)

// Device ...
type Device struct {
	state *syncMap
	name  string
	conn  net.Conn
}

func newDevice(name string, conn net.Conn) *Device {
	return &Device{
		state: newSyncMap(),
		name:  name,
		conn:  conn,
	}
}

// Send a publish packet with topic, payload and QoS 0 to the device
func (d *Device) Send(topic string, payload []byte) error {
	pkg, err := mqtt.MakePublishPacket(topic, payload)
	if err != nil {
		return err
	}
	return pkg.Write(d.conn)
}

// PutState ...
func (d *Device) PutState(key string, value interface{}) {
	d.state.put(key, value)
}

// GetState ...
func (d *Device) GetState(key string) (interface{}, bool) {
	return d.state.get(key)
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

	dev = newDevice(name, conn)
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

type syncMap struct {
	l sync.RWMutex
	m map[string]interface{}
}

func newSyncMap() *syncMap {
	return &syncMap{
		m: make(map[string]interface{}),
	}
}

func (s *syncMap) get(key string) (interface{}, bool) {
	s.l.RLock()
	defer s.l.RUnlock()

	value, exists := s.m[key]
	return value, exists
}

func (s *syncMap) put(key string, value interface{}) {
	s.l.Lock()
	defer s.l.Unlock()

	s.m[key] = value
}

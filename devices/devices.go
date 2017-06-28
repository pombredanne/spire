package devices

import (
	"errors"
	"net"
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

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

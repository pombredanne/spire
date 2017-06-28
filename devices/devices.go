package devices

import (
	"errors"
	"net"
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"fmt"
)

type Device struct {
	name string
	conn net.Conn
}

func NewDevice(name string, conn net.Conn) *Device {
	return &Device{
		name: name,
		conn: conn,
	}
}

func (d *Device) Send(topic string, payload []byte) error {
	pkg := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
	pkg.Qos = 1
	pkg.TopicName = topic
	pkg.Payload = payload
	return pkg.Write(d.conn)
}

var ErrDevNotFound = errors.New("device not found")

type DeviceMap struct {
	l sync.RWMutex
	m map[string]*Device
}

func NewDeviceMap() *DeviceMap {
	return &DeviceMap{
		m: make(map[string]*Device),
	}
}

func (d *DeviceMap) Get(name string) (*Device, error) {
	d.l.RLock()
	defer d.l.RUnlock()
	fmt.Println("GET")
	fmt.Println(d.m)
	dev, exists := d.m[name]
	if !exists {
		return nil, ErrDevNotFound
	}
	return dev, nil
}

func (d *DeviceMap) Add(name string, conn net.Conn) {
	d.l.Lock()
	defer d.l.Unlock()
	fmt.Println("ADD")
	dev, exists := d.m[name]
	if exists {
		if dev.conn == conn {
			return
		}

		dev.conn.Close()
	}
	d.m[name] = NewDevice(name, conn)
}

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

func (d *DeviceMap) Send(name, topic string, payload []byte) error {
	d.l.RLock()
	defer d.l.RUnlock()

	dev, exists := d.m[name]
	if !exists {
		return ErrDevNotFound
	}

	return dev.Send(topic, payload)
}

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

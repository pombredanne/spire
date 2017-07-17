package service

import (
	"log"
	"net"
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/superscale/spire/devices"
)

type state struct {
	FormationState *devices.SyncMap
	Devices        *devices.DeviceMap
}

// Broker manages pub/sub
type Broker struct {
	formations  map[string]state // not sure whether this map needs locking
	subscribers *subscriberMap
}

// NewBroker ...
func NewBroker(devices *devices.DeviceMap) *Broker {
	return &Broker{
		formations:  make(map[string]state),
		subscribers: newSubscriberMap(),
	}
}

// Subscribe adds the connection to the list of subscribers to the topic
func (b *Broker) Subscribe(pkg *packets.SubscribePacket, conn net.Conn) {
	if len(pkg.Topics) > len(pkg.Qoss) {
		// TODO send error reponse to conn
		panic("malformed SUBSCRIBE packet")
	}

	for i, topic := range pkg.Topics {
		if pkg.Qoss[i] > 0 {
			// TODO send error reponse to conn
			panic("QoS > 0 not supported")
		}

		b.subscribers.add(topic, conn)
	}

	sAck := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	sAck.MessageID = pkg.MessageID
	sAck.Write(conn)
}

// Unsubscribe removes the connection from the list of subscribers to the topic
func (b *Broker) Unsubscribe(pkg *packets.UnsubscribePacket, conn net.Conn) {
	for _, topic := range pkg.Topics {
		b.subscribers.removeFromTopic(topic, conn)
	}

	sAck := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	sAck.MessageID = pkg.MessageID
	sAck.Write(conn)
}

// Publish ...
func (b *Broker) Publish(pkg *packets.PublishPacket) {
	// TODO support wildcards
	subs := b.subscribers.get(pkg.TopicName)

	for _, s := range subs {
		// we might want to limit concurrency with a worker pool
		go func(conn net.Conn) {
			err := pkg.Write(conn)
			if err != nil {
				log.Println(err)
			}
		}(s)
	}
	return
}

// Remove ...
func (b *Broker) Remove(conn net.Conn) {
	b.subscribers.remove(conn)
}

type subscriberMap struct {
	l sync.RWMutex
	m map[string][]net.Conn
}

func newSubscriberMap() *subscriberMap {
	return &subscriberMap{
		m: make(map[string][]net.Conn),
	}
}

func (d *subscriberMap) get(topic string) []net.Conn {
	d.l.RLock()
	defer d.l.RUnlock()

	subs, exists := d.m[topic]
	if !exists {
		return []net.Conn{}
	}
	return subs
}

func (d *subscriberMap) add(topic string, conn net.Conn) {
	d.l.Lock()
	defer d.l.Unlock()

	subs, exists := d.m[topic]
	if !exists {
	}

	for _, subConn := range subs {
		if subConn == conn {
			return
		}

		d.m[topic] = append(subs, subConn)
	}

	d.m[topic] = []net.Conn{conn}
}

func (d *subscriberMap) remove(conn net.Conn) {
	d.l.Lock()
	defer d.l.Unlock()

	for topic, conns := range d.m {
		i := indexOf(conns, conn)
		if i < 0 {
			continue
		}

		// from https://github.com/golang/go/wiki/SliceTricks
		copy(conns[i:], conns[i+1:])
		conns[len(conns)-1] = nil
		conns = conns[:len(conns)-1]

		if len(conns) == 0 {
			delete(d.m, topic)
		} else {
			d.m[topic] = conns
		}
	}
}

func (d *subscriberMap) removeFromTopic(topic string, conn net.Conn) {
	d.l.Lock()
	defer d.l.Unlock()

	conns, exists := d.m[topic]
	if !exists {
		return
	}

	// FIXME ugly duplication because of locking
	i := indexOf(conns, conn)
	if i < 0 {
		return
	}

	// from https://github.com/golang/go/wiki/SliceTricks
	copy(conns[i:], conns[i+1:])
	conns[len(conns)-1] = nil
	conns = conns[:len(conns)-1]

	if len(conns) == 0 {
		delete(d.m, topic)
	} else {
		d.m[topic] = conns
	}

}

func indexOf(conns []net.Conn, conn net.Conn) int {
	for i, c := range conns {
		if c == conn {
			return i
		}
	}
	return -1
}

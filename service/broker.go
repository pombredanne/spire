package service

import (
	"log"
	"net"
	"strings"
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
		// according to
		// http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html#_Toc398718110
		// we kill the connection if we don't like a packet
		b.Remove(conn)
		conn.Close()
		return
	}

	for i, topic := range pkg.Topics {
		if pkg.Qoss[i] > 0 {
			b.Remove(conn)
			conn.Close()
			return
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
	subs := []net.Conn{}
	topics := b.subscribers.keys()

	for _, t := range topics {
		subs = append(subs, b.subscribers.get(t)...)
	}

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

// MatchTopics is only exported for tests :(
func MatchTopics(topic string, topics []string) []string {
	matches := []string{}
	topicParts := strings.Split(topic, "/")

	for _, t := range topics {
		if topicsMatch(topicParts, strings.Split(t, "/")) {
			matches = append(matches, t)
		}
	}
	return matches
}

// parameters are the topics split on "/"
// assumes that the topic in the first parameter does not contain wildcards
func topicsMatch(t1, t2 []string) bool {
	l1 := len(t1)
	l2 := len(t2)

	if l1 != l2 && t2[l2-1] != "#" {
		return false
	}

	l := l1
	if l2 < l1 {
		l = l2
	}

	for i := 0; i < l; i++ {
		if t1[i] != t2[i] && t2[i] != "#" {
			return false
		}
	}
	return true
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

func (d *subscriberMap) keys() []string {
	d.l.RLock()
	defer d.l.RUnlock()

	i := 0
	res := make([]string, len(d.m))
	for key := range d.m {
		res[i] = key
		i++
	}

	return res
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

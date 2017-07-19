package mqtt

import (
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// Broker manages pub/sub
type Broker struct {
	l           sync.RWMutex
	subscribers map[string][]net.Conn
}

// NewBroker ...
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]net.Conn),
	}
}

// HandleConnection ...
func (b *Broker) HandleConnection(conn net.Conn) {
	if _, err := Connect(conn); err != nil {
		log.Println(err)
		return
	}

	for {
		ca, err := packets.ReadPacket(conn)
		if err != nil {
			if err != io.EOF {
				log.Println(err)
				b.Remove(conn)
				conn.Close()
			}
			return
		}

		switch ca := ca.(type) {
		case *packets.PublishPacket:
			b.Publish(ca)
		case *packets.SubscribePacket:
			b.Subscribe(ca, conn)
		case *packets.UnsubscribePacket:
			b.Unsubscribe(ca, conn)
		default:
			b.Remove(conn)
			conn.Close()
			return
		}
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

		b.add(topic, conn)
	}

	sAck := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	sAck.MessageID = pkg.MessageID
	sAck.Write(conn)
}

// Unsubscribe removes the connection from the list of subscribers to the topic
func (b *Broker) Unsubscribe(pkg *packets.UnsubscribePacket, conn net.Conn) {
	for _, topic := range pkg.Topics {
		b.removeFromTopic(topic, conn)
	}

	sAck := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	sAck.MessageID = pkg.MessageID
	sAck.Write(conn)
}

// Publish ...
func (b *Broker) Publish(pkg *packets.PublishPacket) {
	topics := MatchTopics(pkg.TopicName, b.topics())
	subs := []net.Conn{}

	for _, t := range topics {
		subs = append(subs, b.get(t)...)
	}

	for _, s := range subs {
		err := pkg.Write(s)
		if err != nil {
			log.Println(err)
		}
	}
}

// Remove ...
func (b *Broker) Remove(conn net.Conn) {
	b.l.Lock()
	defer b.l.Unlock()

	for topic, conns := range b.subscribers {
		i := indexOf(conns, conn)
		if i < 0 {
			continue
		}

		// from https://github.com/golang/go/wiki/SliceTricks
		copy(conns[i:], conns[i+1:])
		conns[len(conns)-1] = nil
		conns = conns[:len(conns)-1]

		if len(conns) == 0 {
			delete(b.subscribers, topic)
		} else {
			b.subscribers[topic] = conns
		}
	}
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

func (b *Broker) topics() []string {
	b.l.RLock()
	defer b.l.RUnlock()

	i := 0
	res := make([]string, len(b.subscribers))
	for topic := range b.subscribers {
		res[i] = topic
		i++
	}

	return res
}

func (b *Broker) get(topic string) []net.Conn {
	b.l.RLock()
	defer b.l.RUnlock()

	subs, exists := b.subscribers[topic]
	if !exists {
		return []net.Conn{}
	}
	return subs
}

func (b *Broker) add(topic string, conn net.Conn) {
	b.l.Lock()
	defer b.l.Unlock()

	subs, exists := b.subscribers[topic]
	if !exists {
	}

	for _, subConn := range subs {
		if subConn == conn {
			return
		}

		b.subscribers[topic] = append(subs, subConn)
	}

	b.subscribers[topic] = []net.Conn{conn}
}

func (b *Broker) removeFromTopic(topic string, conn net.Conn) {
	b.l.Lock()
	defer b.l.Unlock()

	conns, exists := b.subscribers[topic]
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
		delete(b.subscribers, topic)
	} else {
		b.subscribers[topic] = conns
	}

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

func indexOf(conns []net.Conn, conn net.Conn) int {
	for i, c := range conns {
		if c == conn {
			return i
		}
	}
	return -1
}

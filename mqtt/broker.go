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
	Subscribers map[string][]net.Conn
}

// NewBroker ...
func NewBroker() *Broker {
	return &Broker{
		Subscribers: make(map[string][]net.Conn),
	}
}

// HandleConnection ...
func (b *Broker) HandleConnection(conn net.Conn) {
	if _, err := Connect(conn, true); err != nil {
		log.Println(err)
		return
	}

	for {
		pkg, err := packets.ReadPacket(conn)
		if err != nil {
			if err != io.EOF {
				log.Println(err)
				b.Remove(conn)
				conn.Close()
			}
			return
		}

		switch p := pkg.(type) {
		case *packets.PublishPacket:
			b.Publish(p)
		case *packets.SubscribePacket:
			b.Subscribe(p, conn)
		case *packets.UnsubscribePacket:
			b.Unsubscribe(p, conn)
		default:
			b.Remove(conn)
			if err := conn.Close(); err != nil {
				log.Println(err)
			}
			return
		}
	}
}

// Subscribe adds the connection to the list of Subscribers to the topic
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

// Unsubscribe removes the connection from the list of Subscribers to the topic
func (b *Broker) Unsubscribe(pkg *packets.UnsubscribePacket, conn net.Conn) {
	b.l.Lock()
	defer b.l.Unlock()

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
	if len(topics) == 0 {
		return
	}

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

	for topic := range b.Subscribers {
		b.removeFromTopic(topic, conn)
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
	res := make([]string, len(b.Subscribers))
	for topic := range b.Subscribers {
		res[i] = topic
		i++
	}

	return res
}

func (b *Broker) get(topic string) []net.Conn {
	b.l.RLock()
	defer b.l.RUnlock()

	subs, exists := b.Subscribers[topic]
	if !exists {
		return []net.Conn{}
	}
	return subs
}

func (b *Broker) add(topic string, conn net.Conn) {
	b.l.Lock()
	defer b.l.Unlock()

	subs, exists := b.Subscribers[topic]
	if !exists {
	}

	for _, subConn := range subs {
		if subConn == conn {
			return
		}

		b.Subscribers[topic] = append(subs, subConn)
	}

	b.Subscribers[topic] = []net.Conn{conn}
}

// ACHTUNG: caller must acquire and release b.l
func (b *Broker) removeFromTopic(topic string, conn net.Conn) {
	conns, exists := b.Subscribers[topic]
	if !exists {
		return
	}

	i := indexOf(conns, conn)
	if i < 0 {
		return
	}

	// from https://github.com/golang/go/wiki/SliceTricks
	copy(conns[i:], conns[i+1:])
	conns[len(conns)-1] = nil
	conns = conns[:len(conns)-1]

	if len(conns) == 0 {
		delete(b.Subscribers, topic)
	} else {
		b.Subscribers[topic] = conns
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

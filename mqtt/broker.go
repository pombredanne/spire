package mqtt

import (
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/superscale/spire/config"
)

type subscriberMap map[string][]*Conn

// Broker manages pub/sub
type Broker struct {
	l           sync.RWMutex
	Subscribers subscriberMap
	idleTimeout time.Duration
}

// NewBroker ...
func NewBroker() *Broker {
	return &Broker{
		Subscribers: make(subscriberMap),
		idleTimeout: config.Config.IdleConnectionTimeout,
	}
}

// HandleConnection ...
func (b *Broker) HandleConnection(conn *Conn) {
	if _, err := conn.Handshake(); err != nil {
		log.Println(err)
		return
	}

	for {
		pkg, err := conn.Read()
		if err != nil {
			if err != io.EOF {
				log.Println(err)
				b.Remove(conn)
				conn.Close()
			}
			return
		}

		switch p := pkg.(type) {
		case *packets.PingreqPacket:
			err = conn.SendPong()
		case *packets.PublishPacket:
			b.Publish(p)
		case *packets.SubscribePacket:
			err = b.Subscribe(p, conn)
		case *packets.UnsubscribePacket:
			err = b.Unsubscribe(p, conn)
		default:
			b.Remove(conn)
			if err = conn.Close(); err != nil {
				log.Println(err)
			}
			return
		}

		if err != nil {
			log.Printf("error while handling packet in broker. peer %v: %v", conn.RemoteAddr(), err)
		}
	}
}

// Subscribe adds the connection to the list of Subscribers to the topic
func (b *Broker) Subscribe(pkg *packets.SubscribePacket, conn *Conn) error {
	for _, topic := range pkg.Topics {
		b.add(topic, conn)
	}

	sAck := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	sAck.MessageID = pkg.MessageID
	return conn.Write(sAck)
}

// Unsubscribe removes the connection from the list of Subscribers to the topic
func (b *Broker) Unsubscribe(pkg *packets.UnsubscribePacket, conn *Conn) error {
	b.l.Lock()
	defer b.l.Unlock()

	for _, topic := range pkg.Topics {
		b.removeFromTopic(topic, conn)
	}

	sAck := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	sAck.MessageID = pkg.MessageID
	return conn.Write(sAck)
}

// Publish ...
func (b *Broker) Publish(pkg *packets.PublishPacket) {
	topics := MatchTopics(pkg.TopicName, b.topics())
	if len(topics) == 0 {
		return
	}

	subs := []*Conn{}

	for _, t := range topics {
		subs = append(subs, b.get(t)...)
	}

	for _, s := range subs {
		err := s.Write(pkg)
		if err != nil {
			log.Println(err)
		}
	}
}

// Remove ...
func (b *Broker) Remove(conn *Conn) {
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

func (b *Broker) get(topic string) []*Conn {
	b.l.RLock()
	defer b.l.RUnlock()

	subs, exists := b.Subscribers[topic]
	if !exists {
		return []*Conn{}
	}
	return subs
}

func (b *Broker) add(topic string, conn *Conn) {
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

	b.Subscribers[topic] = []*Conn{conn}
}

// ACHTUNG: caller must acquire and release b.l
func (b *Broker) removeFromTopic(topic string, conn *Conn) {
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

func indexOf(conns []*Conn, conn *Conn) int {
	for i, c := range conns {
		if c == conn {
			return i
		}
	}
	return -1
}

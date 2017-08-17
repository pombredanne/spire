package mqtt

import (
	"io"
	"log"
	"strings"
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// PublishHandler defines the function signature for pub/sub
type PublishHandler func(topic string, message interface{}) error

type subscriberMap map[string][]PublishHandler

// Broker manages pub/sub
type Broker struct {
	l           sync.RWMutex
	subscribers subscriberMap
}

// NewBroker ...
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(subscriberMap),
	}
}

// HandleConnection ...
func (b *Broker) HandleConnection(session *Session) {
	if _, err := session.Handshake(); err != nil {
		log.Println(err)
		return
	}

	for {
		pkg, err := session.Read()
		if err != nil {
			if err != io.EOF {
				log.Println(err)
				b.Remove(session.Publish)
				session.Close()
			}
			return
		}

		switch p := pkg.(type) {
		case *packets.PingreqPacket:
			err = session.SendPingresp()
		case *packets.PublishPacket:
			b.Publish(p.TopicName, p.Payload)
		case *packets.SubscribePacket:
			b.SubscribeAll(p, session.Publish)
			err = session.SendSuback(p.MessageID)
		case *packets.UnsubscribePacket:
			b.UnsubscribeAll(p, session.Publish)
			err = session.SendUnsuback(p.MessageID)
		default:
			b.Remove(session.Publish)
			if err = session.Close(); err != nil {
				log.Println(err)
			}
			return
		}

		if err != nil {
			log.Printf("error while handling packet in broker. peer %v: %v", session.RemoteAddr(), err)
		}
	}
}

// Subscribe ...
func (b *Broker) Subscribe(topic string, pubHandler PublishHandler) {
	b.l.Lock()
	defer b.l.Unlock()

	subs, exists := b.subscribers[topic]
	if !exists {
		b.subscribers[topic] = []PublishHandler{pubHandler}
		return
	}

	for _, ph := range subs {
		if &ph == &pubHandler {
			return
		}

		b.subscribers[topic] = append(subs, pubHandler)
	}
}

// SubscribeAll ...
func (b *Broker) SubscribeAll(pkg *packets.SubscribePacket, ph PublishHandler) {
	for _, topic := range pkg.Topics {
		b.Subscribe(topic, ph)
	}
}

// Unsubscribe ...
func (b *Broker) Unsubscribe(topic string, ph PublishHandler) {
	b.l.Lock()
	defer b.l.Unlock()

	subs, exists := b.subscribers[topic]
	if !exists {
		return
	}

	i := indexOf(subs, ph)
	if i < 0 {
		return
	}

	// from https://github.com/golang/go/wiki/SliceTricks
	copy(subs[i:], subs[i+1:])
	subs[len(subs)-1] = nil
	subs = subs[:len(subs)-1]

	if len(subs) == 0 {
		delete(b.subscribers, topic)
	} else {
		b.subscribers[topic] = subs
	}
}

// UnsubscribeAll ...
func (b *Broker) UnsubscribeAll(pkg *packets.UnsubscribePacket, ph PublishHandler) {
	for _, topic := range pkg.Topics {
		b.Unsubscribe(topic, ph)
	}
}

// Publish ...
func (b *Broker) Publish(topic string, message interface{}) {
	topics := MatchTopics(topic, b.topics())
	if len(topics) == 0 {
		return
	}

	handlers := []PublishHandler{}

	for _, t := range topics {
		handlers = append(handlers, b.get(t)...)
	}

	for _, mh := range handlers {
		err := mh(topic, message)
		if err != nil {
			log.Println(err)
		}
	}
}

// Remove ...
func (b *Broker) Remove(mh PublishHandler) {
	for topic := range b.subscribers {
		b.Unsubscribe(topic, mh)
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

func (b *Broker) get(topic string) []PublishHandler {
	b.l.RLock()
	defer b.l.RUnlock()

	subs, exists := b.subscribers[topic]
	if !exists {
		return []PublishHandler{}
	}
	return subs
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

func indexOf(pubHandlers []PublishHandler, pubHandler PublishHandler) int {
	for i, ph := range pubHandlers {
		if &ph == &pubHandler {
			return i
		}
	}
	return -1
}

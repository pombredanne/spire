package testutils

import (
	"net"
	"time"

	"github.com/superscale/spire/mqtt"
)

// Pipe ...
func Pipe() (*mqtt.Session, *mqtt.Session) {
	a, b := net.Pipe()
	t := time.Second * 1
	return mqtt.NewSession(a, t), mqtt.NewSession(b, t)
}

// PubSubRecorder ...
type PubSubRecorder struct {
	Topics   []string
	Messages []interface{}
}

// NewPubSubRecorder ...
func NewPubSubRecorder() *PubSubRecorder {
	return &PubSubRecorder{
		Topics:   []string{},
		Messages: []interface{}{},
	}
}

// Record ...
func (r *PubSubRecorder) Record(topic string, message interface{}) error {
	r.Topics = append(r.Topics, topic)
	r.Messages = append(r.Messages, message)
	return nil
}

// Count ...
func (r *PubSubRecorder) Count() int {
	return len(r.Topics)
}

// Get ...
func (r *PubSubRecorder) Get(i int) (string, interface{}) {
	if i < len(r.Topics) && i < len(r.Messages) {
		return r.Topics[i], r.Messages[i]
	}

	return "", nil
}

// First ...
func (r *PubSubRecorder) First() (string, interface{}) {
	return r.Get(0)
}

// Last ...
func (r *PubSubRecorder) Last() (string, interface{}) {
	return r.Get(r.Count()-1)
}

// HasReceived ...
func (r *PubSubRecorder) HasReceived(topic string, message interface{}) bool {
	for i, t := range r.Topics {
		if t == topic && r.Messages[i] == message {
			return true
		}
	}
	return false
}
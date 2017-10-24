package ota

import (
	"encoding/json"
	"fmt"

	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

type states int

const (
	// Default ...
	Default states = iota
	// Downloading ...
	Downloading
	// Upgrading ...
	Upgrading
	// Error ...
	Error
	// Cancelled ...
	Cancelled
)

// Message ...
type Message struct {
	State    states `json:"state"`    // <"downloading"|"upgrading"|"error">,
	Progress int    `json:"progress"` // <download progress in percent> if state is "downloading"
	Error    string `json:"error"`
	Yours    string `json:"yours"` // if error is "sha256 mismatch"
	Mine     string `json:"mine"`  // if error is "sha256 mismatch"
}

// Handler ...
type Handler struct {
	broker     *mqtt.Broker
	formations *devices.FormationMap
}

// Register ...
func Register(broker *mqtt.Broker, formations *devices.FormationMap) interface{} {
	h := &Handler{broker, formations}

	broker.Subscribe(devices.ConnectTopic.String(), h)
	broker.Subscribe(devices.DisconnectTopic.String(), h)
	broker.Subscribe("pylon/+/ota", h)
	return h
}

// HandleMessage implements mqtt.Subscriber
func (h *Handler) HandleMessage(topic string, message interface{}) error {

	switch t := devices.ParseTopic(topic); t.Path {
	case devices.ConnectTopic.Path:
		return h.onConnect(message.(*devices.ConnectMessage))
	case devices.DisconnectTopic.Path:
		return h.onDisconnect(message.(*devices.DisconnectMessage))
	default:
		buf, ok := message.([]byte)
		if !ok {
			return fmt.Errorf("[OTA] expected byte buffer, got this instead: %v", message)
		}

		msg := new(Message)
		if err := json.Unmarshal(buf, msg); err != nil {
			return err
		}

		return h.onMessage(t, msg)
	}
}

func (h *Handler) onMessage(topic devices.Topic, msg *Message) error {
	formationID := h.formations.FormationID(topic.DeviceName)
	h.formations.PutDeviceState(formationID, topic.DeviceName, "ota", msg)
	h.publish(topic.DeviceName, msg)

	return nil
}

func (h *Handler) onConnect(cm *devices.ConnectMessage) error {
	msg := &Message{State: Default}
	h.formations.PutDeviceState(cm.FormationID, cm.DeviceName, "ota", msg)
	h.publish(cm.DeviceName, msg)

	return nil
}

func (h *Handler) onDisconnect(dm *devices.DisconnectMessage) error {
	rawState := h.formations.GetDeviceState(dm.DeviceName, "ota")
	state, ok := rawState.(*Message)

	if ok && state.State == Downloading {
		h.publish(dm.DeviceName, &Message{State: Error, Error: "connection to device lost during download"})
	}

	return nil
}

func (h *Handler) publish(deviceName string, msg *Message) {
	topic := fmt.Sprintf("armada/%s/ota", deviceName)
	h.broker.Publish(topic, msg)
}

func (s states) String() string {
	switch s {
	case Default:
		return "default"
	case Downloading:
		return "downloading"
	case Upgrading:
		return "upgrading"
	case Error:
		return "error"
	case Cancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// MarshalJSON ...
func (s *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	tmp := &struct {
		State string `json:"state"`
		*Alias
	}{
		State: s.State.String(),
		Alias: (*Alias)(s),
	}
	return json.Marshal(tmp)
}

// UnmarshalJSON ...
func (s *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	tmp := &struct {
		State string `json:"state"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	switch tmp.State {
	case "downloading":
		s.State = Downloading
	case "upgrading":
		s.State = Upgrading
	case "error":
		s.State = Error
	case "cancelled":
		s.State = Cancelled
	default:
		s.State = Default
	}

	return nil
}

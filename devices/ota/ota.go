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
func Register(broker *mqtt.Broker, formations *devices.FormationMap) {
	h := &Handler{broker, formations}

	broker.Subscribe(devices.ConnectTopic, h.onConnect)
	broker.Subscribe(devices.DisconnectTopic, h.onDisconnect)
	broker.Subscribe("/pylon/#/ota", h.onMessage)
}

func (h *Handler) onMessage(topic string, payload interface{}) error {
	buf, ok := payload.([]byte)
	if !ok {
		return fmt.Errorf("[OTA] expected byte buffer, got this instead: %v", payload)
	}

	msg := new(Message)
	if err := json.Unmarshal(buf, msg); err != nil {
		return err
	}

	t := devices.ParseTopic(topic)

	// FIXME We only call this to get the formation ID for the device. It should be part of the topic.
	_, formationID := h.formations.GetDeviceState(t.DeviceName, "ota")

	h.formations.PutDeviceState(formationID, t.DeviceName, "ota", msg)
	h.publish(t.DeviceName, msg)
	return nil
}

func (h *Handler) onConnect(_ string, payload interface{}) error {
	cm := payload.(*devices.ConnectMessage)

	msg := &Message{State: Default}
	h.formations.PutDeviceState(cm.FormationID, cm.DeviceName, "ota", msg)
	h.publish(cm.DeviceName, msg)

	return nil
}

func (h *Handler) onDisconnect(_ string, payload interface{}) error {
	dm := payload.(*devices.DisconnectMessage)

	rawState, _ := h.formations.GetDeviceState(dm.DeviceName, "ota")
	state, ok := rawState.(*Message)

	if ok && state.State == Downloading {
		h.publish(dm.DeviceName, &Message{State: Error, Error: "connection to device lost during download"})
	}

	return nil
}

func (h *Handler) publish(deviceName string, msg *Message) {
	topic := fmt.Sprintf("/armada/%s/ota", deviceName)
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

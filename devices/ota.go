package devices

import (
	"encoding/json"
	"fmt"

	"github.com/superscale/spire/mqtt"
)

type otaStateEnum int

const (
	// Default ...
	Default otaStateEnum = iota
	// Downloading ...
	Downloading
	// Upgrading ...
	Upgrading
	// Error ...
	Error
	// Cancelled ...
	Cancelled
)

// OTAState ...
type OTAState struct {
	State    otaStateEnum `json:"state"`    // <"downloading"|"upgrading"|"error">,
	Progress int          `json:"progress"` // <download progress in percent> if state is "downloading"
	Error    string       `json:"error"`
	Yours    string       `json:"yours"` // if error is "sha256 mismatch"
	Mine     string       `json:"mine"`  // if error is "sha256 mismatch"
}

// HandleOTA ...
func HandleOTA(_ string, payload []byte, formationID, deviceName string, formations *FormationMap, broker *mqtt.Broker) error {
	otaMsg := new(OTAState)
	if err := json.Unmarshal(payload, otaMsg); err != nil {
		return err
	}

	formations.PutDeviceState(formationID, deviceName, "ota", otaMsg)
	return publishOTAMessage(deviceName, otaMsg, broker)
}

func (s otaStateEnum) String() string {
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
func (s *OTAState) MarshalJSON() ([]byte, error) {
	type Alias OTAState
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
func (s *OTAState) UnmarshalJSON(data []byte) error {
	type Alias OTAState
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

func publishOTAMessage(deviceName string, msg *OTAState, broker *mqtt.Broker) error {
	topic := fmt.Sprintf("/armada/%s/ota", deviceName)
	pkg, err := mqtt.MakePublishPacket(topic, msg)
	if err != nil {
		return err
	}

	broker.Publish(pkg)
	return nil
}

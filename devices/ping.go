package devices

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/superscale/spire/mqtt"
)

// PingStats ...
type PingStats struct {
	Sent        int64   `json:"sent"`
	Received    int64   `json:"received"`
	Count       int64   `json:"-"`
	LossNow     float64 `json:"loss_now"`
	Loss24Hours float64 `json:"loss_24_hours"`
}

// PingState ...
type PingState struct {
	Version   int64     `json:"version"`
	Timestamp time.Time `json:"timestamp"`

	Internet struct {
		Ping PingStats `json:"ping"`
		DNS  PingStats `json:"dns"`
	} `json:"internet"`

	Gateway struct {
		Ping PingStats `json:"ping"`
	} `json:"gateway"`

	Tunnel struct {
		Ping PingStats `json:"ping"`
	} `json:"tunnel"`
}

// HandlePing ...
func HandlePing(_ string, payload []byte, formationID, deviceName string, formations *FormationMap, broker *mqtt.Broker) error {
	pingMsg := new(PingState)
	if err := json.Unmarshal(payload, pingMsg); err != nil {
		return err
	}

	currentState, _ := formations.GetDeviceState(formationID, deviceName, "ping").(*PingState)
	newState := updatePingState(currentState, pingMsg)
	formations.PutDeviceState(formationID, deviceName, "ping", newState)

	pkg, err := mqtt.MakePublishPacket(fmt.Sprintf("/armada/%s/wan/ping", deviceName), newState)
	if err != nil {
		return err
	}

	broker.Publish(pkg)
	return nil
}

func updatePingState(currentState, pingMsg *PingState) *PingState {
	if currentState == nil {
		currentState = pingMsg
	}

	resetCount := false
	twelveHoursAgo := time.Now().Add(-12 * time.Hour)
	if currentState.Timestamp.Before(twelveHoursAgo) {
		resetCount = true
		currentState.Timestamp = twelveHoursAgo
	}

	UpdateLosses(&currentState.Internet.Ping, pingMsg.Internet.Ping.Sent, pingMsg.Internet.Ping.Received, resetCount)
	UpdateLosses(&currentState.Internet.DNS, pingMsg.Internet.DNS.Sent, pingMsg.Internet.DNS.Received, resetCount)
	UpdateLosses(&currentState.Gateway.Ping, pingMsg.Gateway.Ping.Sent, pingMsg.Gateway.Ping.Received, resetCount)
	UpdateLosses(&currentState.Tunnel.Ping, pingMsg.Tunnel.Ping.Sent, pingMsg.Tunnel.Ping.Received, resetCount)

	return currentState
}

// UpdateLosses mutates members of the first parameter
func UpdateLosses(stats *PingStats, sent, received int64, resetCount bool) {
	if received == 0 {
		stats.LossNow = 1.0
	} else {
		stats.LossNow = round(1.0-float64(received)/float64(sent), 2)
	}

	stats.Loss24Hours = (stats.Loss24Hours*float64(stats.Count) + stats.LossNow) / float64(stats.Count+1)
	stats.Loss24Hours = round(stats.Loss24Hours, 2)

	stats.Count++

	if resetCount {
		stats.Count = stats.Count / 2
		if stats.Count < 1000 {
			stats.Count = 1000
		}
	}

	stats.Sent = sent
	stats.Received = received
}

func round(f, places float64) float64 {
	shift := math.Pow(10, places)
	f = math.Floor(f*shift + .5)
	return f / shift
}

// MarshalJSON ...
func (ps *PingState) MarshalJSON() ([]byte, error) {
	type Alias PingState
	return json.Marshal(&struct {
		*Alias
		Timestamp int64 `json:"timestamp"`
	}{
		Alias:     (*Alias)(ps),
		Timestamp: ps.Timestamp.Unix(),
	})
}

// UnmarshalJSON ...
func (ps *PingState) UnmarshalJSON(data []byte) error {
	type Alias PingState
	tmp := &struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(ps),
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	ps.Timestamp = time.Unix(tmp.Timestamp, 0)
	return nil
}

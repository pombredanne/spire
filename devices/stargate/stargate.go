package stargate

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

type endpoint struct {
	IP   *string `json:"ip"`
	Port *int    `json:"port"`
}

// http://imgur.com/qni0caW
type assimilator struct {
	Message struct {
		Assimilator interface{} `json:"assimilator"`
	} `json:"message"`
	From endpoint `json:"from"`
}

// PortsMessage ...
type PortsMessage struct {
	TFTPD struct {
		Client       endpoint `json:"client"`
		Listening    *bool    `json:"listening"`
		Request      *string  `json:"request"`
		Total        *int64   `json:"total"`
		Progress     *int64   `json:"progress"`
		Error        *string  `json:"error"`
		Transmitting *int64   `json:"transmitting"`
		Finished     bool     `json:"finished"`
	} `json:"tftpd"`

	Assimilator *assimilator `json:"assimilator"`
	Request     *string      `json:"request"`
	Port        int          `json:"port"`
	Up          interface{}  `json:"up"`
}

const (
	// Key ...
	Key = "stargate"
	// Wait ...
	Wait = "wait"
	// API ...
	API = "api"
	// Start ...
	Start = "start"
	// Transmitting ...
	Transmitting = "transmitting"
	// Progress ...
	Progress = "progress"
	// Flashing ...
	Flashing = "flashing"
	// Assimilator ...
	Assimilator = "assimilator"
	// Ok ...
	Ok = "ok"
	// Error ...
	Error = "error"
)

// PortState ...
type PortState struct {
	Timestamp   int64       `json:"timestamp"`
	State       string      `json:"state"`
	File        string      `json:"file,omitempty"`
	Total       int64       `json:"total,omitempty"`
	Progress    int64       `json:"progress,omitempty"`
	Error       string      `json:"error,omitempty"`
	Assimilator interface{} `json:"assimilator,omitempty"`
}

// NewPortState ...
func NewPortState() *PortState {
	return &PortState{
		Timestamp: time.Now().UTC().Unix(),
		State:     Wait,
	}
}

// PortMap ...
type PortMap map[int]*PortState

// SystemImageMessage ...
type SystemImageMessage struct {
	ID       string `json:"id"`
	Download string `json:"download"`
	Vendor   string `json:"vendor"`
	Product  string `json:"product"`
	Total    int64  `json:"total"`
	Progress int64  `json:"progress"`
}

// SystemImageState ...
type SystemImageState struct {
	ID       string `json:"id"`
	State    string `json:"state"`
	Vendor   string `json:"vendor"`
	Product  string `json:"product"`
	Total    int64  `json:"-"`
	Progress int64  `json:"progress,omitempty"`
}

// NewSystemImageState ...
func NewSystemImageState(id, vendor, product string) *SystemImageState {
	return &SystemImageState{
		ID:      id,
		State:   Start,
		Vendor:  vendor,
		Product: product,
	}
}

// SystemImageMap ...
type SystemImageMap map[string]*SystemImageState

// State ...
type State struct {
	Ports        PortMap
	SystemImages SystemImageMap
}

// NewState ...
func NewState() *State {
	return &State{
		Ports:        make(PortMap),
		SystemImages: make(SystemImageMap),
	}
}

// Handler ...
type Handler struct {
	broker     *mqtt.Broker
	formations *devices.FormationMap
}

// Register ...
func Register(broker *mqtt.Broker, formations *devices.FormationMap) interface{} {
	h := &Handler{
		broker:     broker,
		formations: formations,
	}

	broker.Subscribe("/pylon/#/stargate/port", h.onPortsMessage)
	broker.Subscribe("/pylon/#/stargate/systemimaged", h.onSystemImageMessage)
	return h
}

func (h *Handler) onPortsMessage(topic string, payload interface{}) error {
	buf, ok := payload.([]byte)
	if !ok {
		return fmt.Errorf("[stargate] expected byte buffer, got this instead: %v", payload)
	}

	msg := new(PortsMessage)
	if err := json.Unmarshal(buf, msg); err != nil {
		return err
	}

	deviceName := devices.ParseTopic(topic).DeviceName
	state := h.getState(deviceName)

	if msg.Up != nil || (msg.TFTPD.Listening != nil && *msg.TFTPD.Listening == true) {
		h.handleUp(deviceName, state, msg.Port)
	} else {
		ps := h.getPortState(state, deviceName, msg.Port)

		if msg.TFTPD.Request != nil && msg.TFTPD.Total != nil {
			ps.File = *msg.TFTPD.Request
			ps.Total = *msg.TFTPD.Total
			ps.State = Start

		} else if msg.TFTPD.Transmitting != nil {
			progress := *msg.TFTPD.Transmitting
			ps.Progress = int64(devices.Round(float64(progress)/float64(ps.Total)*100.0, 0))
			ps.State = Transmitting

		} else if msg.TFTPD.Finished == true {
			ps.Progress = 100.0
			ps.State = Flashing

		} else if msg.TFTPD.Error != nil {
			ps.Error = *msg.TFTPD.Error
			ps.State = Error

		} else if msg.Assimilator != nil {
			ps.Assimilator = (*msg.Assimilator).Message.Assimilator
			ps.State = Assimilator
		}
	}

	h.formations.PutDeviceState(h.formations.FormationID(deviceName), deviceName, Key, state)
	h.broker.Publish(fmt.Sprintf("/matriarch/%s/stargate/ports", deviceName), state.Ports)
	return nil
}

func (h *Handler) handleUp(deviceName string, state *State, port int) {
	ps, exists := state.Ports[port]
	if exists {
		ps.State = Wait
		ps.Timestamp = time.Now().UTC().Unix()
	} else {
		state.Ports[port] = NewPortState()
	}
}

func (h *Handler) getState(deviceName string) *State {
	state, ok := h.formations.GetDeviceState(deviceName, Key).(*State)
	if !ok {
		return NewState()
	}
	return state
}

func (h *Handler) getPortState(state *State, deviceName string, port int) *PortState {
	ps, exists := state.Ports[port]
	if !exists {
		log.Printf("[stargate] warning: expected state for port %d on device %s missing", port, deviceName)
		ps = NewPortState()
		state.Ports[port] = ps
	} else {
		ps.Timestamp = time.Now().UTC().Unix()
	}
	return ps
}

func (h *Handler) onSystemImageMessage(topic string, payload interface{}) error {
	buf, ok := payload.([]byte)
	if !ok {
		return fmt.Errorf("[stargate] expected byte buffer, got this instead: %v", payload)
	}

	msg := new(SystemImageMessage)
	if err := json.Unmarshal(buf, msg); err != nil {
		return err
	}

	deviceName := devices.ParseTopic(topic).DeviceName
	state := h.getState(deviceName)

	if msg.Download == API {
		state.SystemImages[msg.ID] = NewSystemImageState(msg.ID, msg.Vendor, msg.Product)
	} else {
		imgState, exists := state.SystemImages[msg.ID]
		if !exists {
			return fmt.Errorf("missing system image state for image %s on device %s", msg.ID, deviceName)
		}

		switch msg.Download {
		case Start:
			imgState.Total = msg.Total
		case Progress:
			imgState.Progress = int64(devices.Round(float64(msg.Progress)/float64(imgState.Total)*100.0, 0))
		case Ok:
			imgState.State = Ok
			imgState.Progress = 100
		default:
			imgState.State = Error
		}
	}

	h.formations.PutDeviceState(h.formations.FormationID(deviceName), deviceName, Key, state)
	h.broker.Publish(fmt.Sprintf("/matriarch/%s/stargate/system_images", deviceName), state.SystemImages)
	return nil
}

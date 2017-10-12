package stations

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

// Key ...
const Key = "stations"

// LanStation ...
type LanStation struct {
	Vendor        string        `json:"vendor"`
	MAC           string        `json:"mac"`
	IP            string        `json:"ip"`
	Port          string        `json:"port"`
	Mode          string        `json:"mode"`
	Local         bool          `json:"local"`
	Age           float64       `json:"age"`
	Seen          int64         `json:"seen"`
	InactiveTime  time.Duration `json:"inactive_time"`
	LastUpdatedAt time.Time     `json:"-"`
}

// Thing ...
type Thing struct {
	Vendor        string                 `json:"vendor"`
	MAC           string                 `json:"mac"`
	IP            string                 `json:"ip"`
	Port          string                 `json:"port"`
	Mode          string                 `json:"mode"`
	Local         bool                   `json:"local"`
	Age           float64                `json:"age"`
	Seen          int64                  `json:"seen"`
	InactiveTime  time.Duration          `json:"inactive_time"`
	LastUpdatedAt time.Time              `json:"-"`
	Thing         map[string]interface{} `json:"thing"`
}

// State contains all data the stations handler needs to compile messages
// for the support frontend.
type State struct {
	WifiStations map[string]WifiStation // MAC -> WifiStation
	LanStations  map[string]*LanStation // MAC -> LanStation
	Things       map[string]*Thing      // IP -> Thing
}

// NewState ...
func NewState() *State {
	return &State{
		WifiStations: make(map[string]WifiStation),
		LanStations:  make(map[string]*LanStation),
		Things:       make(map[string]*Thing),
	}
}

// Message is the data structure published to the MQTT bus for control/UI clients to consume.
type Message struct {
	Public  []WifiStation `json:"public"`
	Private []WifiStation `json:"private"`
	Other   []*LanStation `json:"other"`
	Thing   []*Thing      `json:"thing"`
}

// Handler ...
type Handler struct {
	broker     *mqtt.Broker
	formations *devices.FormationMap
}

// Register ...
func Register(broker *mqtt.Broker, formations *devices.FormationMap) interface{} {
	h := &Handler{broker: broker, formations: formations}

	broker.Subscribe("/pylon/+/wifi/poll", h.onWifiPollMessage)
	broker.Subscribe("/pylon/+/wifi/event", h.onWifiEventMessage)
	broker.Subscribe("/pylon/+/things/discovery", h.onThingsMessage)
	broker.Subscribe("/pylon/+/net", h.onNetMessage)
	broker.Subscribe("/pylon/+/sys/facts", h.onSysMessage)
	broker.Subscribe("/pylon/+/odhcpd", h.onDHCPMessage)
	return h
}

func (h *Handler) onWifiPollMessage(topic string, payload interface{}) error {
	msg, err := unmarshalWifiPollMessage(payload)
	if err != nil {
		return err
	}

	surveyMsg, err := compileWifiSurveyMessage(msg)
	if err != nil {
		return err
	}

	deviceName := devices.ParseTopic(topic).DeviceName
	state, formationID := h.getState(deviceName)

	h.updateWifiStations(msg, state, deviceName)

	h.formations.PutState(formationID, Key, state)

	surveyTopic := fmt.Sprintf("/matriarch/%s/wifi/survey", deviceName)
	h.broker.Publish(surveyTopic, surveyMsg)

	h.publish(deviceName, state)
	return nil
}

func (h *Handler) onWifiEventMessage(topic string, payload interface{}) error {
	msg, err := unmarshalWifiEventMessage(payload)
	if err != nil {
		return err
	}

	deviceName := devices.ParseTopic(topic).DeviceName
	state, formationID := h.getState(deviceName)

	if msg.Action == "assoc" {
		state.WifiStations[msg.MAC] = WifiStation{"mac": msg.MAC}
	} else if msg.Action == "disassoc" {
		delete(state.WifiStations, msg.MAC)
	}

	h.formations.PutState(formationID, Key, state)
	h.publish(deviceName, state)
	return nil
}

func (h *Handler) updateWifiStations(msg *WifiPollMessage, state *State, deviceName string) {

	for ifaceName, iface := range msg.Interfaces {

		stations, err := ParseWifiStations(iface.Stations, ifaceName)

		if err == nil {
			state.WifiStations = merge(state.WifiStations, stations)
		} else {
			log.Printf("[stations] error while parsing wifi station info for interface %s on device %s: %v", ifaceName, deviceName, err)
		}
	}
}

func (h *Handler) getState(deviceName string) (*State, string) {
	formationID := h.formations.FormationID(deviceName)
	state, ok := h.formations.GetState(formationID, Key).(*State)
	if !ok {
		return NewState(), formationID
	}

	return state, formationID
}

func (h *Handler) onThingsMessage(topic string, payload interface{}) error {
	msg, err := unmarshalThingsMessage(payload)
	if err != nil {
		return err
	}

	ip, ipOk := msg["address"].(string)
	t, tOk := msg["thing"].(map[string]interface{})
	if !ipOk || !tOk {
		return fmt.Errorf("[stations] got invalid things discovery message: %v", msg)
	}

	deviceName := devices.ParseTopic(topic).DeviceName
	state, formationID := h.getState(deviceName)

	thing, exists := state.Things[ip]
	if exists {
		thing.Thing = t
		thing.LastUpdatedAt = time.Now().UTC()
	} else {
		thing := &Thing{
			IP:            ip,
			LastUpdatedAt: time.Now().UTC(),
			Thing:         t,
			Mode:          "thing",
		}

		state.Things[ip] = thing
	}

	h.formations.PutState(formationID, Key, state)
	h.publish(deviceName, state)
	return nil
}

type netMessage struct {
	MAC []struct {
		MAC string `json:"mac"`
		IP  string `json:"ip"`
	} `json:"mac"`

	Bridge struct {
		MACs struct {
			Public  string `json:"public"`
			Private string `json:"private"`
		} `json:"macs"`
	} `json:"bridge"`

	Switch string `json:"switch"`
}

func (h *Handler) onNetMessage(topic string, payload interface{}) error {
	msg, err := unmarshalNetMessage(payload)
	if err != nil {
		return err
	}

	deviceName := devices.ParseTopic(topic).DeviceName
	state, formationID := h.getState(deviceName)
	now := time.Now().UTC()

	for _, e := range msg.MAC {
		if thing := state.Things[e.IP]; thing != nil {
			thing.MAC = e.MAC
			thing.Vendor = vendorFromMAC(e.MAC)
			thing.LastUpdatedAt = now
		} else {
			if station, exists := state.WifiStations[e.MAC]; exists {
				station["ip"] = e.IP
			} else {
				state.LanStations[e.MAC] = &LanStation{
					Vendor:        vendorFromMAC(e.MAC),
					MAC:           e.MAC,
					IP:            e.IP,
					Mode:          "other",
					LastUpdatedAt: now,
				}
			}
		}
	}

	if err := h.assignPorts(msg, deviceName, state); err != nil {
		log.Printf("[stations] error while assigning ports from switch info for device %s: %v", deviceName, err)
	}

	if err := h.assignBridgeInfo(msg, deviceName, state); err != nil {
		log.Printf("[stations] error while assigning bridge info for device %s: %v", deviceName, err)
	}

	h.removeTimedOutStations(state)
	h.formations.PutState(formationID, Key, state)
	h.publish(deviceName, state)
	return nil
}

const lanStationTimeout = time.Minute * 10
const thingTimeout = time.Minute * 5

func (h *Handler) removeTimedOutStations(state *State) {
	now := time.Now().UTC()

	for mac, ls := range state.LanStations {
		ls.InactiveTime = now.Sub(ls.LastUpdatedAt)
		if ls.InactiveTime > lanStationTimeout {
			delete(state.LanStations, mac)
		}
	}

	for ip, thing := range state.Things {
		thing.InactiveTime = now.Sub(thing.LastUpdatedAt)
		if thing.InactiveTime > thingTimeout {
			delete(state.Things, ip)
		}
	}
}

func (h *Handler) assignPorts(msg *netMessage, deviceName string, state *State) error {
	var mac2port map[string]string
	var err error
	cpuPorts, ok := h.formations.GetDeviceState(deviceName, "cpu_ports").([]string)
	if ok {
		_, mac2port, err = ParseSwitch(msg.Switch, cpuPorts...)
	} else {
		_, mac2port, err = ParseSwitch(msg.Switch)
	}
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for mac, ws := range state.WifiStations {
		if port, exists := mac2port[mac]; exists {
			ws["port"] = port
		}
	}

	for mac, ls := range state.LanStations {
		if port, exists := mac2port[mac]; exists {
			ls.Port = port
			ls.LastUpdatedAt = now
		}
	}

	for _, t := range state.Things {
		if len(t.MAC) == 0 {
			continue
		}

		if port, exists := mac2port[t.MAC]; exists {
			t.Port = port
			t.LastUpdatedAt = now
		}
	}

	return nil
}

func (h *Handler) assignBridgeInfo(msg *netMessage, deviceName string, state *State) error {
	bridgeInfo, err := ParseBridgeMACs(msg.Bridge.MACs.Private)
	if err != nil {
		return err
	}
	pubBI, err := ParseBridgeMACs(msg.Bridge.MACs.Public)
	if err != nil {
		return err
	}
	for m, bi := range pubBI {
		bridgeInfo[m] = bi
	}

	now := time.Now().UTC()
	for mac, ws := range state.WifiStations {
		if bi, exists := bridgeInfo[mac]; exists {
			ws["age"] = bi.Age
			ws["local"] = bi.Local
		}
	}

	for mac, ls := range state.LanStations {
		if bi, exists := bridgeInfo[mac]; exists {
			ls.Age = bi.Age
			ls.Local = bi.Local
			ls.LastUpdatedAt = now
		}
	}

	for _, t := range state.Things {
		if len(t.MAC) == 0 {
			continue
		}

		if bi, exists := bridgeInfo[t.MAC]; exists {
			t.Age = bi.Age
			t.Local = bi.Local
			t.LastUpdatedAt = now
		}
	}

	return nil
}

type sysMessage struct {
	Board struct {
		Switch struct {
			Switch0 struct {
				Ports []struct {
					Number int     `json:"num"`
					Device *string `json:"device"`
				} `json:"ports"`
			} `json:"switch0"`
		} `json:"switch"`
	} `json:"board"`
}

func (h *Handler) onSysMessage(topic string, payload interface{}) error {
	msg, err := unmarshalSysMessage(payload)
	if err != nil {
		return err
	}

	cpuPorts := []string{}
	for _, port := range msg.Board.Switch.Switch0.Ports {
		if port.Device != nil {
			cpuPorts = append(cpuPorts, strconv.Itoa(port.Number))
		}
	}

	deviceName := devices.ParseTopic(topic).DeviceName
	h.formations.PutDeviceState(h.formations.FormationID(deviceName), deviceName, "cpu_ports", cpuPorts)
	return nil
}

func (h *Handler) onDHCPMessage(topic string, payload interface{}) error {
	buf, ok := payload.([]byte)
	if !ok {
		return fmt.Errorf("[stations] expected byte buffer, got this instead: %v", payload)
	}

	dhcpState, err := ParseDHCP(buf)
	if err != nil {
		return err
	}

	deviceName := devices.ParseTopic(topic).DeviceName
	h.broker.Publish(fmt.Sprintf("/matriarch/%s/dhcp/leases", deviceName), dhcpState)
	return nil
}

func round(f float64) int64 {
	return int64(math.Floor(f + 0.5))
}

func (h *Handler) publish(deviceName string, state *State) {
	now := time.Now().UTC().Unix()

	msg := &Message{
		Public:  []WifiStation{},
		Private: []WifiStation{},
		Other:   make([]*LanStation, len(state.LanStations)),
		Thing:   []*Thing{},
	}

	for _, station := range state.WifiStations {
		if age, ok := station["age"].(float64); ok {
			station["seen"] = now - round(age)
		}

		if station["mode"] == "public" {
			msg.Public = append(msg.Public, station)
		} else {
			msg.Private = append(msg.Private, station)
		}
	}

	for _, thing := range state.Things {
		if len(thing.MAC) > 0 {
			thing.Seen = now - round(thing.Age)
			msg.Thing = append(msg.Thing, thing)
		}
	}

	i := 0
	for _, station := range state.LanStations {
		station.Seen = now - round(station.Age)
		msg.Other[i] = station
		i++
	}

	h.broker.Publish(fmt.Sprintf("/matriarch/%s/stations", deviceName), msg)
}

func unmarshalWifiPollMessage(payload interface{}) (*WifiPollMessage, error) {
	buf, ok := payload.([]byte)
	if !ok {
		return nil, fmt.Errorf("[stations] expected byte buffer, got this instead: %v", payload)
	}

	msg := new(WifiPollMessage)
	if err := json.Unmarshal(buf, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func unmarshalWifiEventMessage(payload interface{}) (*WifiEventMessage, error) {
	buf, ok := payload.([]byte)
	if !ok {
		return nil, fmt.Errorf("[stations] expected byte buffer, got this instead: %v", payload)
	}

	msg := new(WifiEventMessage)
	if err := json.Unmarshal(buf, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func unmarshalThingsMessage(payload interface{}) (map[string]interface{}, error) {
	buf, ok := payload.([]byte)
	if !ok {
		return nil, fmt.Errorf("[stations] expected byte buffer, got this instead: %v", payload)
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(buf, &msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func unmarshalNetMessage(payload interface{}) (*netMessage, error) {
	buf, ok := payload.([]byte)
	if !ok {
		return nil, fmt.Errorf("[stations] expected byte buffer, got this instead: %v", payload)
	}

	msg := new(netMessage)
	if err := json.Unmarshal(buf, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func unmarshalSysMessage(payload interface{}) (*sysMessage, error) {
	buf, ok := payload.([]byte)
	if !ok {
		return nil, fmt.Errorf("[stations] expected byte buffer, got this instead: %v", payload)
	}

	msg := new(sysMessage)
	if err := json.Unmarshal(buf, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

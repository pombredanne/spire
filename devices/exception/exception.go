package exception

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bugsnag/bugsnag-go"
	"github.com/superscale/spire/config"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

// Message ...
type Message struct {
	Error   string `json:"error"`
	Context string `json:"context"`
}

// Handler ...
type Handler struct {
	formations *devices.FormationMap
}

// Register ...
func Register(broker *mqtt.Broker, formations *devices.FormationMap) interface{} {
	h := &Handler{formations}
	broker.Subscribe("/pylon/+/exception", h.onMessage)
	return h
}

func (h *Handler) onMessage(topic string, payload interface{}) error {
	if len(config.Config.BugsnagKey) == 0 {
		return errors.New("bugsnag API key not set")
	}

	buf, ok := payload.([]byte)
	if !ok {
		return fmt.Errorf("[exception] expected byte buffer, got this instead: %v", payload)
	}

	m := Message{Error: "unknown exception on device", Context: "unknown originating topic"}
	if err := json.Unmarshal(buf, &m); err != nil {
		return err
	}

	t := devices.ParseTopic(topic)
	metadata := bugsnag.MetaData{}
	metadata.Add("device", "hostname", t.DeviceName)

	rawState := h.formations.GetDeviceState(t.DeviceName, "device_info")

	if deviceInfo, ok := rawState.(map[string]interface{}); ok {
		metadata.Add("device", "osVersion", deviceInfo["device_os"])
	} else {
		metadata.Add("device", "osVersion", "unknown")
	}

	return bugsnag.Notify(errors.New(m.Error), bugsnag.SeverityError, bugsnag.Context{String: m.Context}, metadata)
}

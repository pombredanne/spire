package devices

import (
	"encoding/json"
	"errors"

	"github.com/bugsnag/bugsnag-go"
	"github.com/superscale/spire/config"
	"github.com/superscale/spire/mqtt"
)

// HandleException ...
func HandleException(_ string, payload []byte, formationID, deviceName string, formations *FormationMap, _ *mqtt.Broker) error {
	if len(config.Config.BugsnagKey) == 0 {
		return errors.New("bugsnag API key not set")
	}

	message := make(map[string]interface{})
	if err := json.Unmarshal(payload, &message); err != nil {
		return err
	}

	var deviceError string
	var ok bool
	deviceError, ok = message["error"].(string)
	if !ok {
		deviceError = "unknown exception on device"
	}

	var ctx string
	ctx, ok = message["context"].(string)
	if !ok {
		ctx = "unknown originating topic"
	}

	metadata := bugsnag.MetaData{}
	metadata.Add("device", "hostname", deviceName)

	if deviceInfo, ok := formations.GetDeviceState(formationID, deviceName, "device_info").(map[string]interface{}); ok {
		metadata.Add("device", "osVersion", deviceInfo["device_os"])
	} else {
		metadata.Add("device", "osVersion", "unknown")
	}

	return bugsnag.Notify(errors.New(deviceError), bugsnag.SeverityError, bugsnag.Context{String: ctx}, metadata)
}

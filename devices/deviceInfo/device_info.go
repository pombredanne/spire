package deviceInfo

import (
	"fmt"

	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

// Handler ...
type Handler struct {
	formations *devices.FormationMap
}

// Register ...
func Register(broker *mqtt.Broker, formations *devices.FormationMap) interface{} {
	h := &Handler{formations: formations}
	broker.Subscribe(devices.ConnectTopic.String(), h)
	return h
}

// HandleMessage ...
func (h *Handler) HandleMessage(_ string, message interface{}) error {
	h.formations.Lock()
	defer h.formations.Unlock()

	cm := message.(devices.ConnectMessage)
	state := map[string]interface{}{"device_os": getDeviceOS(cm.DeviceInfo)}
	h.formations.PutDeviceState(cm.FormationID, cm.DeviceName, "device_info", state)
	return nil
}

func getDeviceOS(info map[string]interface{}) (res string) {
	res = "unknown"

	data, ok := info["data"].(map[string]interface{})
	if ok {
		sysimg, ok := data["current_system_image"].(map[string]interface{})
		if ok {
			vendor, ok := sysimg["vendor"].(string)
			if !ok {
				return
			}
			product, ok := sysimg["product"].(string)
			if !ok {
				return
			}
			variant, ok := sysimg["variant"].(string)
			if !ok {
				return
			}
			version, ok := sysimg["version"].(float64)
			if !ok {
				return
			}
			res = fmt.Sprintf("%s-%s-%s-%d", vendor, product, variant, int(version))
		}
	}

	return
}

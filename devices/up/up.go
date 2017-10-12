package up

import (
	"context"
	"fmt"
	"time"

	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

// Handler ...
type Handler struct {
	broker     *mqtt.Broker
	formations *devices.FormationMap
}

// Register ...
func Register(broker *mqtt.Broker, formations *devices.FormationMap) interface{} {
	h := &Handler{broker, formations}

	broker.Subscribe(devices.ConnectTopic, h.onConnect)
	broker.Subscribe(devices.DisconnectTopic, h.onDisconnect)
	return h
}

func (h *Handler) onConnect(_ string, payload interface{}) error {
	cm := payload.(*devices.ConnectMessage)

	ctx, cancelFn := context.WithCancel(context.Background())
	h.formations.PutDeviceState(cm.FormationID, cm.DeviceName, "cancelUpFn", cancelFn)

	go h.publishUpState(ctx, cm.DeviceName)
	return nil
}

func (h *Handler) onDisconnect(_ string, payload interface{}) error {
	cm := payload.(*devices.DisconnectMessage)

	r := h.formations.GetDeviceState(cm.DeviceName, "cancelUpFn")
	cancelFn, ok := r.(context.CancelFunc)
	if !ok {
		return fmt.Errorf("cannot cancel goroutine that publishes 'up' state for device %s", cm.DeviceName)
	}

	cancelFn()

	h.formations.DeleteDeviceState(cm.FormationID, cm.DeviceName, "cancelUpFn")
	return nil
}

func (h *Handler) publishUpState(ctx context.Context, deviceName string) {
	topic := fmt.Sprintf("armada/%s/up", deviceName)

	msg := map[string]interface{}{
		"state":     "up",
		"timestamp": time.Now().UTC().Unix(),
	}

	h.broker.Publish(topic, msg)

	for {
		select {
		case <-ctx.Done():
			msg["state"] = "down"
			h.broker.Publish(topic, msg)
			return
		case <-time.After(30 * time.Second):
			h.broker.Publish(topic, msg)
		}
	}
}

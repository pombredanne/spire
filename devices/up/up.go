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

	broker.Subscribe(devices.ConnectTopic.String(), h)
	broker.Subscribe(devices.DisconnectTopic.String(), h)
	return h
}

// HandleMessage ...
func (h *Handler) HandleMessage(topic string, message interface{}) error {
	h.formations.Lock()
	defer h.formations.Unlock()

	switch t := devices.ParseTopic(topic); t.Path {
	case devices.ConnectTopic.Path:
		return h.onConnect(message.(devices.ConnectMessage))
	case devices.DisconnectTopic.Path:
		return h.onDisconnect(message.(devices.DisconnectMessage))
	default:
		return nil
	}
}

func (h *Handler) onConnect(cm devices.ConnectMessage) error {
	ctx, cancelFn := context.WithCancel(context.Background())
	h.formations.PutDeviceState(cm.FormationID, cm.DeviceName, "cancelUpFn", cancelFn)

	go h.publishUpState(ctx, cm.DeviceName)
	return nil
}

func (h *Handler) onDisconnect(dm devices.DisconnectMessage) error {
	r := h.formations.GetDeviceState(dm.DeviceName, "cancelUpFn")
	cancelFn, ok := r.(context.CancelFunc)
	if !ok {
		return fmt.Errorf("cannot cancel goroutine that publishes 'up' state for device %s", dm.DeviceName)
	}

	cancelFn()

	h.formations.DeleteDeviceState(dm.FormationID, dm.DeviceName, "cancelUpFn")
	return nil
}

func (h *Handler) publishUpState(ctx context.Context, deviceName string) {
	topic := fmt.Sprintf("matriarch/%s/up", deviceName)

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

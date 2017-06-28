package control

import (
	"net"

	"github.com/superscale/spire/devices"
)

type ControlMessageHandler struct {
	devices *devices.DeviceMap
}

func NewControlMessageHandler(devices *devices.DeviceMap) *ControlMessageHandler {
	return &ControlMessageHandler{
		devices: devices,
	}
}

func (h *ControlMessageHandler) HandleConnection(conn net.Conn) {
	// TODO
}

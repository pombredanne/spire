package handlers

import (
	"log"
	"github.com/superscale/spire/devices"
)

// HandlePing ...
func HandlePing(deviceName, topic string, payload []byte, devices *devices.DeviceMap) error {
	log.Println("It's a ping!")
	return nil
}

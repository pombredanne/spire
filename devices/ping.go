package devices

import (
	"log"
)

// HandlePing ...
func HandlePing(deviceName, topic string, payload []byte, devices *DeviceMap) error {
	log.Println("It's a ping!")
	return nil
}

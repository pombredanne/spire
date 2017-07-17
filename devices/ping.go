package devices

import (
	"log"

	"github.com/superscale/spire/mqtt"
)

// HandlePing ...
func HandlePing(deviceName, topic string, payload []byte, state *State, broker *mqtt.Broker) error {
	log.Println("It's a ping!")
	return nil
}

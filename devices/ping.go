package devices

import (
	"log"

	"github.com/superscale/spire/service"
)

// HandlePing ...
func HandlePing(deviceName, topic string, payload []byte, state *State, broker *service.Broker) error {
	log.Println("It's a ping!")
	return nil
}

package devices

import (
	"log"

	"github.com/superscale/spire/mqtt"
)

// HandlePing ...
func HandlePing(topic string, payload []byte, formation *formationS, broker *mqtt.Broker) error {
	log.Println("It's a ping!")
	return nil
}

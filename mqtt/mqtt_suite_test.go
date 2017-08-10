package mqtt_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/config"
)

var _ = BeforeSuite(func() {
	config.Config.Environment = "test"
})

// TestHandlers ...
func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spire MQTT Suite")
}

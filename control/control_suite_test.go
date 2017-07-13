package control_test

import (

	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/superscale/spire/service"
)

var _ = BeforeSuite(func() {
	service.Config.Environment = "test"
})

// TestControl ...
func TestControl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spire Control Suite")
}

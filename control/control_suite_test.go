package control_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestControl ...
func TestControl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spire Control Suite")
}

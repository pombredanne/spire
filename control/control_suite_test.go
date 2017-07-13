package control_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/redis.v5"
	"github.com/superscale/spire/service"
)

var redisClient redis.Cmdable

var _ = BeforeSuite(func() {
	service.Config.Environment = "test"
	service.Config.RedisURL = "redis://redis:6379"

	var err error
	redisClient, err = service.InitRedis()
	Expect(err).NotTo(HaveOccurred())
})

// TestControl ...
func TestControl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spire Control Suite")
}

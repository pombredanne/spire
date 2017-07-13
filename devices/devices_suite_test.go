package devices_test

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

// TestHandlers ...
func TestHandlers(t *testing.T) {
	BeforeEach(func() {
		var err error
		_, err = redisClient.FlushAll().Result()
		Expect(err).NotTo(HaveOccurred())
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "Spire Handlers Suite")
}

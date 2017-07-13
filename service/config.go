package service

import (
	"log"

	"gopkg.in/redis.v5"
)

// ConfigS defines all possible config params
type ConfigS struct {
	Environment    string `env:"ENV"  envDefault:"prod"`
	DevicesBind    string `env:"DEVICES_BIND"  envDefault:":1883"`
	ServicesBind   string `env:"SERVICES_BIND"  envDefault:":1884"`
	RedisURL       string `env:"REDIS_URL,required"`
	RedisClustered bool   `env:"REDIS_CLUSTERED" envDefault:"true"`
}

// Config is the global handle for accessing runtime configuration
var Config = &ConfigS{}

// InitRedis ...
func InitRedis() (redisClient redis.Cmdable, err error) {
	redopts, err := redis.ParseURL(Config.RedisURL)
	if err != nil {
		return
	}

	log.Println("connecting to redis " + Config.RedisURL)
	if Config.RedisClustered {
		redisClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    []string{redopts.Addr},
			Password: redopts.Password,
		})
	} else {
		redisClient = redis.NewClient(redopts)
	}
	_, err = redisClient.Ping().Result()
	if err != nil {
		return
	}
	return
}

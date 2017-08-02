package service

import (
	"github.com/bugsnag/bugsnag-go"
	consul "github.com/hashicorp/consul/api"
	"strings"
	"strconv"
)

// ConfigS defines all possible config params
type ConfigS struct {
	Environment       string `env:"ENV"  envDefault:"prod"`
	DevicesBind       string `env:"DEVICES_BIND"  envDefault:":1883"`
	ControlBind       string `env:"CONTROL_BIND"  envDefault:":1884"`
	BugsnagKey        string `env:"BUGSNAG_KEY"`
	LiberatorBaseURL  string `env:"LIBERATOR_BASE_URL"  envDefault:"https://api.superscale.io"`
	LiberatorJWTToken string `env:"LIBERATOR_JWT_TOKEN,required"`
}

// Config is the global handle for accessing runtime configuration
var Config = &ConfigS{}

// InitBugsnag ...
func InitBugsnag() {
	if len(Config.BugsnagKey) > 0 {
		bugsnag.Configure(bugsnag.Configuration{
			APIKey:       Config.BugsnagKey,
			ReleaseStage: Config.Environment,
		})
	}
}

const serviceName = "spire"
const controlTag = "control"
const devicesTag = "devices"

// RegisterInConsul ...
func RegisterInConsul() error {
	config := consul.DefaultConfig()
	client, err := consul.NewClient(config)
	if err != nil {
		return err
	}

	if err := registerServiceInConsul(client, serviceName, controlTag, Config.ControlBind); err != nil {
		return err
	}

	return registerServiceInConsul(client, serviceName, devicesTag, Config.DevicesBind)
}

func registerServiceInConsul(client *consul.Client, serviceName, serviceTag, bind string) error {
	registration := &consul.AgentServiceRegistration{
		Name: serviceName,
		Tags: []string{serviceTag},
	}

	port, err := strconv.Atoi(strings.Split(bind, ":")[1])
	if err != nil {
		return err
	}
	registration.Port = port

	return client.Agent().ServiceRegister(registration)
}

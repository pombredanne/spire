package config

import (
	"time"

	"github.com/caarlos0/env"
)

// Params defines all possible config params
type Params struct {
	Environment           string        `env:"ENV"  envDefault:"prod"`
	DevicesBind           string        `env:"DEVICES_BIND"  envDefault:":1883"`
	ControlBind           string        `env:"CONTROL_BIND"  envDefault:":1884"`
	BugsnagKey            string        `env:"BUGSNAG_KEY"`
	LiberatorBaseURL      string        `env:"LIBERATOR_BASE_URL"  envDefault:"https://api.superscale.io"`
	LiberatorJWTToken     string        `env:"LIBERATOR_JWT_TOKEN,required"`
	IdleConnectionTimeout time.Duration `env:"IDLE_CONNECTION_TIMEOUT"  envDefault:"30s"`
}

// Config is the global handle for accessing runtime configuration
var Config = &Params{}

// Parse environment variables into Config
func Parse() {
	err := env.Parse(Config)
	if err != nil {
		panic(err)
	}
}

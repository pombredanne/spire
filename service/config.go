package service

import "github.com/bugsnag/bugsnag-go"

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

package service

// ConfigS defines all possible config params
type ConfigS struct {
	Environment  string `env:"ENV"  envDefault:"prod"`
	DevicesBind  string `env:"DEVICES_BIND"  envDefault:":1883"`
	ServicesBind string `env:"SERVICES_BIND"  envDefault:":1884"`
}

// Config is the global handle for accessing runtime configuration
var Config = &ConfigS{}

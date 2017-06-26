package service

// ConfigS defines all possible config params
type ConfigS struct {
	Environment string `env:"ENV"  envDefault:"prod"`
	Bind        string `env:"BIND"  envDefault:":1883"`
}

// Config is the global handle for accessing runtime configuration
var Config = &ConfigS{}

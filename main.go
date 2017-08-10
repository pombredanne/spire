package main

import (
	"github.com/bugsnag/bugsnag-go"
	"github.com/superscale/spire/config"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
)

func main() {
	config.Parse()

	if len(config.Config.BugsnagKey) > 0 {
		bugsnag.Configure(bugsnag.Configuration{
			APIKey:       config.Config.BugsnagKey,
			ReleaseStage: config.Config.Environment,
		})
	}

	broker := mqtt.NewBroker()

	devMsgHandler := devices.NewMessageHandler(broker)
	devicesServer := mqtt.NewServer(config.Config.DevicesBind, devMsgHandler.HandleConnection)
	go devicesServer.Run()

	controlServer := mqtt.NewServer(config.Config.ControlBind, broker.HandleConnection)
	controlServer.Run()
}

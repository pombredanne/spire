package main

import (
	"github.com/bugsnag/bugsnag-go"
	"github.com/superscale/spire/config"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/devices/deviceInfo"
	"github.com/superscale/spire/devices/exception"
	"github.com/superscale/spire/devices/ota"
	"github.com/superscale/spire/devices/ping"
	"github.com/superscale/spire/devices/up"
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
	formations := devices.NewFormationMap()
	loadMessageHandlers(broker, formations)

	devHandler := devices.NewHandler(broker)
	devicesServer := mqtt.NewServer(config.Config.DevicesBind, devHandler.HandleConnection)
	go devicesServer.Run()

	controlServer := mqtt.NewServer(config.Config.ControlBind, broker.HandleConnection)
	controlServer.Run()
}

type registerFn func(*mqtt.Broker, *devices.FormationMap)

func loadMessageHandlers(broker *mqtt.Broker, formations *devices.FormationMap) {

	regFns := []registerFn{
		deviceInfo.Register,
		exception.Register,
		ota.Register,
		ping.Register,
		up.Register,
	}

	for _, register := range regFns {
		register(broker, formations)
	}
}

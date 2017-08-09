package main

import (
	"github.com/caarlos0/env"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/mqtt"
	"github.com/superscale/spire/service"
)

func main() {
	err := env.Parse(service.Config)
	if err != nil {
		panic(err)
	}

	service.InitBugsnag()

	broker := mqtt.NewBroker()

	devMsgHandler := devices.NewMessageHandler(broker)
	devicesServer := service.NewServer(service.Config.DevicesBind, devMsgHandler.HandleConnection)
	go devicesServer.Run()

	controlServer := service.NewServer(service.Config.ControlBind, broker.HandleConnection)
	controlServer.Run()
}

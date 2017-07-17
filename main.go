package main

import (
	"os"
	"os/signal"

	"github.com/caarlos0/env"
	"github.com/superscale/spire/service"
	"github.com/superscale/spire/devices"
	"github.com/superscale/spire/control"
)

func main() {
	err := env.Parse(service.Config)
	if err != nil {
		panic(err)
	}

	state := devices.NewState()
	broker := service.NewBroker()

	devMsgHandler := devices.NewMessageHandler(state, broker)

	devicesServer := service.NewServer(service.Config.DevicesBind, devMsgHandler.HandleConnection)
	devicesServer.Run()

	ctrlMsgHandler := control.NewMessageHandler(state, broker)
	controlServer := service.NewServer(service.Config.ServicesBind, ctrlMsgHandler.HandleConnection)
	controlServer.Run()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit

	devicesServer.Shutdown()
	controlServer.Shutdown()
}

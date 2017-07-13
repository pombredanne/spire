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

	redisClient, err := service.InitRedis()
	if err != nil {
		panic(err)
	}

	devs := devices.NewDeviceMap()
	devMsgHandler := devices.NewMessageHandler(devs, redisClient)

	devicesServer := service.NewServer(service.Config.DevicesBind, devMsgHandler.HandleConnection)
	devicesServer.Run()

	ctrlMsgHandler := control.NewMessageHandler(devs, redisClient)
	controlServer := service.NewServer(service.Config.ServicesBind, ctrlMsgHandler.HandleConnection)
	controlServer.Run()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit

	devicesServer.Shutdown()
	controlServer.Shutdown()
}

package main

import (
	"github.com/caarlos0/env"
	"github.com/superscale/spire/service"
)

func main() {
	err := env.Parse(service.Config)
	if err != nil {
		panic(err)
	}

	server := service.Server{}
	if err := server.Run(); err != nil {
		panic(err)
	}
}

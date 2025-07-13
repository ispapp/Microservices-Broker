package main

import (
	"log"
	"os"

	"github.com/ispapp/Microservices-Broker/cmd"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:           "Microservices Broker",
		Usage:          "Simple Microservices Broker",
		DefaultCommand: "serve",
		Commands: []*cli.Command{
			cmd.ServerCommand,
			cmd.ConfigCommand,
			cmd.AuthCommand,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

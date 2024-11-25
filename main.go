package main

import (
	"Microservices-Broker/cmd"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "Microservices Broker",
		Usage: "Simple Microservices Broker",
		Commands: []*cli.Command{
			cmd.ServerCommand,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

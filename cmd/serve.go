package cmd

import (
	"Microservices-Broker/base/pb"
	"Microservices-Broker/cmd/lib"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

var ServerCommand = &cli.Command{
	Name:  "serve",
	Usage: "Start the Microservices Broker server",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "input",
			Aliases: []string{"i"},
			Usage:   "Input db folder (broker.db: bitcask)",
			Value:   "broker.db",
		},
		&cli.StringFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "Port to serve on",
			Value:   "50011",
		},
	},
	Action: func(c *cli.Context) error {
		port := c.String("port")
		lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", port))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		input := c.String("input")
		server, err := lib.NewServer(input, 60, 100, time.Hour*24)
		if err != nil {
			log.Fatalf("failed to create server: %v", err)
		}
		s := grpc.NewServer()
		pb.RegisterBrokerServer(s, server)
		log.Printf("server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
		return nil
	},
}

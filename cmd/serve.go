package cmd

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/kmoz000/Microservices-Broker/base/pb"
	"github.com/kmoz000/Microservices-Broker/cmd/lib"

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
		&cli.StringFlag{
			Name:    "host",
			Aliases: []string{"H"},
			Usage:   "Host to listen on",
			Value:   "0.0.0.0",
		},
	},
	Action: func(c *cli.Context) error {
		port := c.String("port")
		host := c.String("host")
		lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", host, port))
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

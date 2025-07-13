package cmd

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/ispapp/Microservices-Broker/base/pb"
	"github.com/ispapp/Microservices-Broker/cmd/lib"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Configuration file path",
			Value:   "config.json",
		},
		&cli.BoolFlag{
			Name:  "disable-auth",
			Usage: "Disable authentication (not recommended for production)",
			Value: false,
		},
	},
	Action: func(c *cli.Context) error {
		configPath := c.String("config")
		disableAuth := c.Bool("disable-auth")

		// Load configuration
		config, err := lib.LoadConfig(configPath)
		if err != nil {
			log.Printf("Warning: Failed to load config file, using defaults: %v", err)
			config = &lib.Config{
				Server: lib.ServerConfig{
					Host:        c.String("host"),
					Port:        c.String("port"),
					TLSEnabled:  false,
					TickSeconds: 60,
					MaxStored:   100,
					MaxAge:      time.Hour * 24,
				},
				Auth: lib.AuthConfig{
					EnableAuth:  !disableAuth,
					AuthMethod:  lib.AuthMethodJWT,
					TokenExpiry: time.Hour * 24,
					APIKeys:     make(map[string]string),
				},
				DB: lib.DBConfig{
					Path: c.String("input"),
				},
			}
		}

		// Override with command line flags if provided
		if c.IsSet("host") {
			config.Server.Host = c.String("host")
		}
		if c.IsSet("port") {
			config.Server.Port = c.String("port")
		}
		if c.IsSet("input") {
			config.DB.Path = c.String("input")
		}
		if disableAuth {
			config.Auth.EnableAuth = false
		}

		// Initialize authentication manager
		authManager := lib.NewAuthManager(&config.Auth)

		// Create server
		server, err := lib.NewServer(config.DB.Path, config.Server.TickSeconds, config.Server.MaxStored, config.Server.MaxAge)
		if err != nil {
			log.Fatalf("failed to create server: %v", err)
		}

		// Setup listener
		lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		// Configure gRPC server options
		var opts []grpc.ServerOption

		// Add authentication interceptors
		if config.Auth.EnableAuth {
			opts = append(opts,
				grpc.UnaryInterceptor(authManager.UnaryInterceptor()),
				grpc.StreamInterceptor(authManager.StreamInterceptor()),
			)
			log.Printf("Authentication enabled (method: %d)", config.Auth.AuthMethod)
		} else {
			log.Printf("WARNING: Authentication is disabled!")
		}

		// Add TLS if enabled
		if config.Server.TLSEnabled {
			cert, err := tls.LoadX509KeyPair(config.Server.TLSCertFile, config.Server.TLSKeyFile)
			if err != nil {
				log.Fatalf("failed to load TLS credentials: %v", err)
			}
			creds := credentials.NewTLS(&tls.Config{
				Certificates: []tls.Certificate{cert},
			})
			opts = append(opts, grpc.Creds(creds))
			log.Printf("TLS enabled")
		}

		// Create gRPC server
		s := grpc.NewServer(opts...)
		pb.RegisterBrokerServer(s, server)

		log.Printf("Microservices Broker server listening at %v", lis.Addr())
		log.Printf("Database path: %s", config.DB.Path)
		log.Printf("Configuration: %s", configPath)

		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
		return nil
	},
}

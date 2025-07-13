package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/ispapp/Microservices-Broker/cmd/lib"
	"github.com/urfave/cli/v2"
)

var AuthCommand = &cli.Command{
	Name:  "auth",
	Usage: "Authentication management commands",
	Subcommands: []*cli.Command{
		{
			Name:  "generate-key",
			Usage: "Generate a new API key for a service",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "service",
					Aliases:  []string{"s"},
					Usage:    "Service name",
					Required: true,
				},
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "Configuration file path",
					Value:   "config.json",
				},
			},
			Action: func(c *cli.Context) error {
				serviceName := c.String("service")
				configPath := c.String("config")

				config, err := lib.LoadConfig(configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}

				authManager := lib.NewAuthManager(&config.Auth)
				apiKey := authManager.GenerateAPIKey(serviceName)

				// Save the updated config
				if err := config.SaveConfig(configPath); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}

				fmt.Printf("Generated API key for service '%s': %s\n", serviceName, apiKey)
				return nil
			},
		},
		{
			Name:  "generate-jwt",
			Usage: "Generate a JWT token for a service",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "service",
					Aliases:  []string{"s"},
					Usage:    "Service name",
					Required: true,
				},
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "Configuration file path",
					Value:   "config.json",
				},
			},
			Action: func(c *cli.Context) error {
				serviceName := c.String("service")
				configPath := c.String("config")

				config, err := lib.LoadConfig(configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}

				authManager := lib.NewAuthManager(&config.Auth)
				token, err := authManager.GenerateJWT(serviceName)
				if err != nil {
					return fmt.Errorf("failed to generate JWT: %w", err)
				}

				fmt.Printf("Generated JWT token for service '%s': %s\n", serviceName, token)
				return nil
			},
		},
		{
			Name:  "list-keys",
			Usage: "List all API keys and their associated services",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "Configuration file path",
					Value:   "config.json",
				},
			},
			Action: func(c *cli.Context) error {
				configPath := c.String("config")

				config, err := lib.LoadConfig(configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}

				if len(config.Auth.APIKeys) == 0 {
					fmt.Println("No API keys found")
					return nil
				}

				fmt.Println("API Keys:")
				fmt.Println("=========")
				for key, service := range config.Auth.APIKeys {
					fmt.Printf("Service: %s\nAPI Key: %s\n\n", service, key)
				}
				return nil
			},
		},
		{
			Name:  "remove-key",
			Usage: "Remove an API key",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "key",
					Aliases:  []string{"k"},
					Usage:    "API key to remove",
					Required: true,
				},
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "Configuration file path",
					Value:   "config.json",
				},
			},
			Action: func(c *cli.Context) error {
				apiKey := c.String("key")
				configPath := c.String("config")

				config, err := lib.LoadConfig(configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}

				if serviceName, exists := config.Auth.APIKeys[apiKey]; exists {
					delete(config.Auth.APIKeys, apiKey)
					if err := config.SaveConfig(configPath); err != nil {
						return fmt.Errorf("failed to save config: %w", err)
					}
					fmt.Printf("Removed API key for service '%s'\n", serviceName)
				} else {
					fmt.Println("API key not found")
				}
				return nil
			},
		},
		{
			Name:  "provision-broker-yaml",
			Usage: "Provision or update a YAML config for another service with broker name and key (multi-service, auto-generate key if missing)",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Aliases:  []string{"n"},
					Usage:    "Broker service name",
					Required: true,
				},
				&cli.StringFlag{
					Name:    "key",
					Aliases: []string{"k"},
					Usage:   "Broker key (optional, will generate if missing)",
				},
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Usage:   "Output YAML file path",
					Value:   "config.yml",
				},
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "Broker JSON config file for key lookup/generation",
					Value:   "config.json",
				},
			},
			Action: func(c *cli.Context) error {
				name := c.String("name")
				key := c.String("key")
				output := c.String("output")
				configPath := c.String("config")
				var authConfig *lib.AuthConfig
				var cfg *lib.Config
				var err error
				if configPath != "" {
					cfg, err = lib.LoadConfig(configPath)
					if err == nil {
						authConfig = &cfg.Auth
					}
				}
				finalKey, err := lib.WriteOrUpdateBrokerKeyYAMLWithAutoKey(output, name, key, authConfig)
				if err != nil {
					return fmt.Errorf("failed to write/update YAML config: %w", err)
				}
				// Save the updated config
				if err := cfg.SaveConfig(configPath); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				fmt.Printf("Provisioned/updated broker YAML config at %s for service '%s' with key: %s\n", output, name, finalKey)
				return nil
			},
		},
	},
}

var ConfigCommand = &cli.Command{
	Name:  "config",
	Usage: "Configuration management commands",
	Subcommands: []*cli.Command{
		{
			Name:  "init-config",
			Usage: "Initialize a default configuration file",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "Configuration file path",
					Value:   "config.json",
				},
			},
			Action: func(c *cli.Context) error {
				configPath := c.String("config")

				// Check if file already exists
				if _, err := os.Stat(configPath); err == nil {
					fmt.Printf("Configuration file '%s' already exists\n", configPath)
					return nil
				}

				if err := lib.GenerateDefaultConfig(configPath); err != nil {
					return fmt.Errorf("failed to generate config: %w", err)
				}

				fmt.Printf("Generated default configuration file: %s\n", configPath)
				return nil
			},
		},
		{
			Name:  "show",
			Usage: "Show current configuration",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "Configuration file path",
					Value:   "config.json",
				},
			},
			Action: func(c *cli.Context) error {
				configPath := c.String("config")

				config, err := lib.LoadConfig(configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}

				fmt.Printf("Server Configuration:\n")
				fmt.Printf("  Host: %s\n", config.Server.Host)
				fmt.Printf("  Port: %s\n", config.Server.Port)
				fmt.Printf("  TLS Enabled: %t\n", config.Server.TLSEnabled)
				fmt.Printf("  TLS Cert File: %s\n", config.Server.TLSCertFile)
				fmt.Printf("  TLS Key File: %s\n", config.Server.TLSKeyFile)
				fmt.Printf("  Tick Seconds: %d\n", config.Server.TickSeconds)
				fmt.Printf("  Max Stored: %d\n", config.Server.MaxStored)
				fmt.Printf("  Max Age: %s\n", config.Server.MaxAge)

				fmt.Printf("\nAuthentication Configuration:\n")
				fmt.Printf("  Enabled: %t\n", config.Auth.EnableAuth)
				fmt.Printf("  Method: %d (0=JWT, 1=API Key)\n", config.Auth.AuthMethod)
				fmt.Printf("  Number of API Keys: %d\n", len(config.Auth.APIKeys))

				fmt.Printf("\nDatabase Configuration:\n")
				fmt.Printf("  Path: %s\n", config.DB.Path)

				return nil
			},
		},
		{
			Name:  "set-auth-method",
			Usage: "Set authentication method (jwt or apikey)",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "method",
					Aliases:  []string{"m"},
					Usage:    "Authentication method (jwt or apikey)",
					Required: true,
				},
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "Configuration file path",
					Value:   "config.json",
				},
			},
			Action: func(c *cli.Context) error {
				method := c.String("method")
				configPath := c.String("config")

				config, err := lib.LoadConfig(configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}

				switch method {
				case "jwt":
					config.Auth.AuthMethod = lib.AuthMethodJWT
				case "apikey":
					config.Auth.AuthMethod = lib.AuthMethodAPIKey
				default:
					return fmt.Errorf("invalid authentication method: %s (use 'jwt' or 'apikey')", method)
				}

				if err := config.SaveConfig(configPath); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}

				fmt.Printf("Authentication method set to: %s\n", method)
				return nil
			},
		},
		{
			Name:  "enable-tls",
			Usage: "Enable TLS for the server",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "cert",
					Usage: "TLS certificate file path",
					Value: "server.crt",
				},
				&cli.StringFlag{
					Name:  "key",
					Usage: "TLS key file path",
					Value: "server.key",
				},
				&cli.StringFlag{
					Name:    "config",
					Aliases: []string{"c"},
					Usage:   "Configuration file path",
					Value:   "config.json",
				},
			},
			Action: func(c *cli.Context) error {
				certFile := c.String("cert")
				keyFile := c.String("key")
				configPath := c.String("config")

				config, err := lib.LoadConfig(configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}

				config.Server.TLSEnabled = true
				config.Server.TLSCertFile = certFile
				config.Server.TLSKeyFile = keyFile

				if err := config.SaveConfig(configPath); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}

				fmt.Printf("TLS enabled with cert: %s, key: %s\n", certFile, keyFile)
				log.Printf("Note: Make sure the certificate and key files exist and are properly configured")
				return nil
			},
		},
	},
}

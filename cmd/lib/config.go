package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the broker configuration
type Config struct {
	Server ServerConfig `json:"server"`
	Auth   AuthConfig   `json:"auth"`
	DB     DBConfig     `json:"database"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host        string        `json:"host"`
	Port        string        `json:"port"`
	TLSEnabled  bool          `json:"tls_enabled"`
	TLSCertFile string        `json:"tls_cert_file"`
	TLSKeyFile  string        `json:"tls_key_file"`
	TickSeconds int16         `json:"tick_seconds"`
	MaxStored   int32         `json:"max_stored"`
	MaxAge      time.Duration `json:"max_age"`
}

// DBConfig holds database-specific configuration
type DBConfig struct {
	Path string `json:"path"`
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	// Default configuration
	config := &Config{
		Server: ServerConfig{
			Host:        "0.0.0.0",
			Port:        "50011",
			TLSEnabled:  false,
			TickSeconds: 60,
			MaxStored:   100,
			MaxAge:      time.Hour * 24,
		},
		Auth: AuthConfig{
			EnableAuth: true,
			AuthMethod: AuthMethodJWT,
			APIKeys:    make(map[string]string),
		},
		DB: DBConfig{
			Path: "broker.db",
		},
	}

	// Load from file if exists
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}

			if err := json.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	return config, nil
}

// SaveConfig saves configuration to file
func (c *Config) SaveConfig(configPath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GenerateDefaultConfig creates a default configuration file
func GenerateDefaultConfig(configPath string) error {
	config := &Config{
		Server: ServerConfig{
			Host:        "0.0.0.0",
			Port:        "50011",
			TLSEnabled:  false,
			TLSCertFile: "server.crt",
			TLSKeyFile:  "server.key",
			TickSeconds: 60,
			MaxStored:   100,
			MaxAge:      time.Hour * 24,
		},
		Auth: AuthConfig{
			EnableAuth: true,
			AuthMethod: AuthMethodJWT,
			JWTSecret:  generateRandomKey(32),
			APIKeys: map[string]string{
				generateRandomKey(32): "service-1",
				generateRandomKey(32): "service-2",
			},
		},
		DB: DBConfig{
			Path: "broker.db",
		},
	}

	return config.SaveConfig(configPath)
}

// BrokerYAMLConfig represents the YAML config structure for broker provisioning
// Example:
// broker:
//
//	name: "service_name"
//	key: "broker_key"
type BrokerYAMLConfig struct {
	Broker struct {
		Name string `yaml:"name"`
		Key  string `yaml:"key"`
	} `yaml:"broker"`
}

// WriteBrokerKeyYAML writes the broker name and key to a YAML file
func WriteBrokerKeyYAML(filePath, name, key string) error {
	cfg := BrokerYAMLConfig{}
	cfg.Broker.Name = name
	cfg.Broker.Key = key
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0600)
}

// MultiBrokerYAMLConfig represents multiple brokers for YAML provisioning
// brokers:
//   - name: "service1"
//     key: "key1"
//   - name: "service2"
//     key: "key2"
type MultiBrokerYAMLConfig struct {
	Brokers []struct {
		Name string `yaml:"name"`
		Key  string `yaml:"key"`
	} `yaml:"brokers"`
}

// WriteOrUpdateBrokerKeyYAML adds or updates a broker entry in the YAML file
func WriteOrUpdateBrokerKeyYAML(filePath, name, key string) error {
	var cfg MultiBrokerYAMLConfig
	// Try to read existing file
	if data, err := os.ReadFile(filePath); err == nil {
		yaml.Unmarshal(data, &cfg)
	}
	// Check if broker exists
	updated := false
	for i, b := range cfg.Brokers {
		if b.Name == name {
			cfg.Brokers[i].Key = key
			updated = true
			break
		}
	}
	if !updated {
		cfg.Brokers = append(cfg.Brokers, struct {
			Name string `yaml:"name"`
			Key  string `yaml:"key"`
		}{Name: name, Key: key})
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0600)
}

// WriteOrUpdateBrokerKeyYAMLWithAutoKey adds/updates a service key in YAML, generating a key if missing
// Only the 'services' key is updated; all other YAML content is preserved.
func WriteOrUpdateBrokerKeyYAMLWithAutoKey(filePath, name, key string, authConfig *AuthConfig) (string, error) {
	// Read existing YAML as a generic map
	var root map[string]interface{}
	if data, err := os.ReadFile(filePath); err == nil {
		yaml.Unmarshal(data, &root)
	} else {
		root = make(map[string]interface{})
	}

	// If key is empty, check AuthConfig or generate
	if key == "" && authConfig != nil {
		for k, svc := range authConfig.APIKeys {
			if svc == name {
				key = k
				break
			}
		}
		if key == "" {
			am := NewAuthManager(authConfig)
			key = am.GenerateAPIKey(name)
		}
	}

	// Update or create the 'services' map
	services, ok := root["services"].(map[string]interface{})
	if !ok {
		services = make(map[string]interface{})
	}
	services[name] = key
	root["services"] = services

	// Marshal and write back
	data, err := yaml.Marshal(root)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", err
	}
	return key, nil
}

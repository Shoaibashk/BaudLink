/*
Copyright 2024 BaudLink Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package config provides configuration loading and management for BaudLink agent.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the complete agent configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	TLS     TLSConfig     `yaml:"tls"`
	Serial  SerialConfig  `yaml:"serial"`
	Logging LoggingConfig `yaml:"logging"`
	Service ServiceConfig `yaml:"service"`
	Metrics MetricsConfig `yaml:"metrics"`
}

// ServerConfig holds server-related settings
type ServerConfig struct {
	GRPCAddress       string `yaml:"grpc_address"`
	WebSocketAddress  string `yaml:"websocket_address"`
	WebSocketEnabled  bool   `yaml:"websocket_enabled"`
	MaxConnections    int    `yaml:"max_connections"`
	ConnectionTimeout int    `yaml:"connection_timeout"`
}

// TLSConfig holds TLS/SSL settings
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

// SerialConfig holds serial port settings
type SerialConfig struct {
	Defaults          SerialDefaults `yaml:"defaults"`
	ScanInterval      int            `yaml:"scan_interval"`
	ExcludePatterns   []string       `yaml:"exclude_patterns"`
	AllowSharedAccess bool           `yaml:"allow_shared_access"`
}

// SerialDefaults holds default serial port parameters
type SerialDefaults struct {
	BaudRate       int    `yaml:"baud_rate"`
	DataBits       int    `yaml:"data_bits"`
	StopBits       int    `yaml:"stop_bits"`
	Parity         string `yaml:"parity"`
	FlowControl    string `yaml:"flow_control"`
	ReadTimeoutMs  int    `yaml:"read_timeout_ms"`
	WriteTimeoutMs int    `yaml:"write_timeout_ms"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// ServiceConfig holds system service settings
type ServiceConfig struct {
	Name          string `yaml:"name"`
	DisplayName   string `yaml:"display_name"`
	Description   string `yaml:"description"`
	AutoStart     bool   `yaml:"auto_start"`
	RestartPolicy string `yaml:"restart_policy"`
	RestartDelay  int    `yaml:"restart_delay"`
}

// MetricsConfig holds metrics/monitoring settings
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
	Path    string `yaml:"path"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			GRPCAddress:       "0.0.0.0:50051",
			WebSocketAddress:  "0.0.0.0:8080",
			WebSocketEnabled:  false,
			MaxConnections:    100,
			ConnectionTimeout: 30,
		},
		TLS: TLSConfig{
			Enabled: false,
		},
		Serial: SerialConfig{
			Defaults: SerialDefaults{
				BaudRate:       9600,
				DataBits:       8,
				StopBits:       1,
				Parity:         "none",
				FlowControl:    "none",
				ReadTimeoutMs:  1000,
				WriteTimeoutMs: 1000,
			},
			ScanInterval:      5,
			AllowSharedAccess: false,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     30,
			Compress:   true,
		},
		Service: ServiceConfig{
			Name:          "baudlink",
			DisplayName:   "BaudLink Serial Agent",
			Description:   "Cross-platform serial port background service",
			AutoStart:     true,
			RestartPolicy: "on-failure",
			RestartDelay:  5,
		},
		Metrics: MetricsConfig{
			Enabled: false,
			Address: "0.0.0.0:9090",
			Path:    "/metrics",
		},
	}
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// LoadOrDefault loads configuration from file, or returns default if file doesn't exist
func LoadOrDefault(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	return Load(path)
}

// Save writes configuration to a YAML file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.GRPCAddress == "" {
		return fmt.Errorf("grpc_address is required")
	}

	if c.Server.MaxConnections < 1 {
		return fmt.Errorf("max_connections must be at least 1")
	}

	if c.TLS.Enabled {
		if c.TLS.CertFile == "" || c.TLS.KeyFile == "" {
			return fmt.Errorf("TLS cert_file and key_file are required when TLS is enabled")
		}
	}

	if c.Serial.Defaults.BaudRate < 1 {
		return fmt.Errorf("baud_rate must be positive")
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[strings.ToLower(c.Logging.Level)] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	return nil
}

// applyEnvOverrides applies environment variable overrides
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("BAUDLINK_GRPC_ADDRESS"); v != "" {
		c.Server.GRPCAddress = v
	}
	if v := os.Getenv("BAUDLINK_LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := os.Getenv("BAUDLINK_TLS_ENABLED"); v == "true" {
		c.TLS.Enabled = true
	}
	if v := os.Getenv("BAUDLINK_TLS_CERT"); v != "" {
		c.TLS.CertFile = v
	}
	if v := os.Getenv("BAUDLINK_TLS_KEY"); v != "" {
		c.TLS.KeyFile = v
	}
}

// DefaultConfigPath returns the default configuration file path for the current OS
func DefaultConfigPath() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("ProgramData"), "BaudLink", "agent.yaml")
	case "darwin":
		return "/usr/local/etc/baudlink/agent.yaml"
	default:
		return "/etc/baudlink/agent.yaml"
	}
}

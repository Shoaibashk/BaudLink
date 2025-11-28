//go:build linux || darwin

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

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Shoaibashk/BaudLink/config"
)

const systemdServiceTemplate = `[Unit]
Description={{.Description}}
After=network.target

[Service]
Type=simple
ExecStart={{.ExecPath}} serve --config {{.ConfigPath}}
Restart={{.RestartPolicy}}
RestartSec={{.RestartDelay}}
User={{.User}}
Group={{.Group}}
WorkingDirectory={{.WorkingDirectory}}

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths={{.LogPath}} {{.ConfigDir}}

# Resource limits
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`

// SystemdService represents a systemd service configuration
type SystemdService struct {
	config  *config.Config
	startFn func() error
	stopFn  func()
}

// NewSystemdService creates a new systemd service
func NewSystemdService(cfg *config.Config, startFn func() error, stopFn func()) *SystemdService {
	return &SystemdService{
		config:  cfg,
		startFn: startFn,
		stopFn:  stopFn,
	}
}

// Run runs the service (directly, not via systemd)
func (ss *SystemdService) Run() error {
	fmt.Println("Running in foreground mode. Press Ctrl+C to stop.")
	return ss.startFn()
}

// serviceData holds data for the systemd template
type serviceData struct {
	Name             string
	Description      string
	ExecPath         string
	ConfigPath       string
	ConfigDir        string
	LogPath          string
	WorkingDirectory string
	User             string
	Group            string
	RestartPolicy    string
	RestartDelay     int
}

// Install installs the systemd service
func Install(cfg *config.Config) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get absolute path
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	configPath := GetConfigPath()
	configDir := filepath.Dir(configPath)
	logPath := GetLogPath()

	// Ensure directories exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.MkdirAll(logPath, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Copy config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := cfg.Save(configPath); err != nil {
			fmt.Printf("Warning: failed to save config: %v\n", err)
		}
	}

	data := serviceData{
		Name:             cfg.Service.Name,
		Description:      cfg.Service.Description,
		ExecPath:         exePath,
		ConfigPath:       configPath,
		ConfigDir:        configDir,
		LogPath:          logPath,
		WorkingDirectory: "/",
		User:             "root", // Could be configurable
		Group:            "root",
		RestartPolicy:    convertRestartPolicy(cfg.Service.RestartPolicy),
		RestartDelay:     cfg.Service.RestartDelay,
	}

	// Parse and execute template
	tmpl, err := template.New("systemd").Parse(systemdServiceTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Write service file
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", cfg.Service.Name)
	f, err := os.Create(servicePath)
	if err != nil {
		return fmt.Errorf("failed to create service file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable service if auto-start is configured
	if cfg.Service.AutoStart {
		if err := runCommand("systemctl", "enable", cfg.Service.Name); err != nil {
			fmt.Printf("Warning: failed to enable service: %v\n", err)
		}
	}

	fmt.Printf("Service %s installed successfully\n", cfg.Service.Name)
	fmt.Printf("  Config: %s\n", configPath)
	fmt.Printf("  Logs: %s\n", logPath)
	fmt.Println()
	fmt.Println("To start the service:")
	fmt.Printf("  sudo systemctl start %s\n", cfg.Service.Name)
	fmt.Println()
	fmt.Println("To check status:")
	fmt.Printf("  sudo systemctl status %s\n", cfg.Service.Name)

	return nil
}

// Uninstall removes the systemd service
func Uninstall(cfg *config.Config) error {
	// Stop the service first
	_ = Stop(cfg)

	// Disable the service
	_ = runCommand("systemctl", "disable", cfg.Service.Name)

	// Remove service file
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", cfg.Service.Name)
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	fmt.Printf("Service %s removed successfully\n", cfg.Service.Name)
	return nil
}

// Start starts the systemd service
func Start(cfg *config.Config) error {
	if err := runCommand("systemctl", "start", cfg.Service.Name); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	fmt.Printf("Service %s started\n", cfg.Service.Name)
	return nil
}

// Stop stops the systemd service
func Stop(cfg *config.Config) error {
	if err := runCommand("systemctl", "stop", cfg.Service.Name); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	fmt.Printf("Service %s stopped\n", cfg.Service.Name)
	return nil
}

// Status returns the status of the systemd service
func Status(cfg *config.Config) (string, error) {
	out, err := exec.Command("systemctl", "is-active", cfg.Service.Name).Output()
	if err != nil {
		// is-active returns exit code 3 for inactive/failed
		status := strings.TrimSpace(string(out))
		if status == "" {
			return "not installed", nil
		}
		return status, nil
	}
	return strings.TrimSpace(string(out)), nil
}

// GetConfigPath returns the config path for Linux/macOS
func GetConfigPath() string {
	return "/etc/baudlink/agent.yaml"
}

// GetLogPath returns the log path for Linux/macOS
func GetLogPath() string {
	return "/var/log/baudlink"
}

// runCommand runs a command and returns an error if it fails
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// convertRestartPolicy converts our restart policy to systemd format
func convertRestartPolicy(policy string) string {
	switch strings.ToLower(policy) {
	case "always":
		return "always"
	case "on-failure":
		return "on-failure"
	case "never":
		return "no"
	default:
		return "on-failure"
	}
}

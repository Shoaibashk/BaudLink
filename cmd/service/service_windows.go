//go:build windows

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

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Shoaibashk/BaudLink/config"
	svc "github.com/Shoaibashk/BaudLink/service"
)

// serviceCmd represents the service command
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the BaudLink Windows service",
	Long: `Manage the BaudLink agent as a Windows service.

This command allows you to install, uninstall, start, stop, and check the
status of the BaudLink agent running as a Windows service.

Subcommands:
  install   - Install the Windows service
  uninstall - Remove the Windows service
  start     - Start the Windows service
  stop      - Stop the Windows service
  status    - Check the Windows service status`,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Windows service and start system tray",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServiceConfig(cmd)
		if err != nil {
			return err
		}

		// Install the service
		if err := svc.Install(cfg); err != nil {
			return err
		}

		fmt.Printf("Service %s installed successfully\n", cfg.Service.Name)

		// Auto-start the service
		fmt.Println("Starting service...")
		if err := svc.Start(cfg); err != nil {
			fmt.Printf("Warning: failed to auto-start service: %v\n", err)
			fmt.Println("You can start it manually with: baudlink-service service start")
		} else {
			fmt.Printf("Service %s started successfully\n", cfg.Service.Name)

			// Start tray application
			if err := startTrayApp(); err != nil {
				fmt.Printf("Warning: failed to start tray application: %v\n", err)
				fmt.Println("You can manually start the tray with: baudlink-tray.exe")
			} else {
				fmt.Println("System tray application started")
			}
		}

		return nil
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the Windows service",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServiceConfig(cmd)
		if err != nil {
			return err
		}

		// Stop tray application first
		if err := stopTrayApp(); err != nil {
			fmt.Printf("Warning: failed to stop tray application: %v\n", err)
		}

		return svc.Uninstall(cfg)
	},
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Windows service and system tray",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServiceConfig(cmd)
		if err != nil {
			return err
		}

		// Start the Windows service
		if err := svc.Start(cfg); err != nil {
			return err
		}
		fmt.Printf("Service %s started successfully\n", cfg.Service.Name)

		// Start tray application
		if err := startTrayApp(); err != nil {
			fmt.Printf("Warning: failed to start tray application: %v\n", err)
			fmt.Println("You can manually start the tray with: baudlink-tray.exe")
		} else {
			fmt.Println("System tray application started")
		}

		return nil
	},
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Windows service and system tray",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServiceConfig(cmd)
		if err != nil {
			return err
		}

		// Stop tray application first
		if err := stopTrayApp(); err != nil {
			fmt.Printf("Warning: failed to stop tray application: %v\n", err)
		} else {
			fmt.Println("System tray application stopped")
		}

		// Stop the Windows service
		if err := svc.Stop(cfg); err != nil {
			return err
		}
		fmt.Printf("Service %s stopped successfully\n", cfg.Service.Name)

		return nil
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the Windows service status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServiceConfig(cmd)
		if err != nil {
			return err
		}
		status, err := svc.Status(cfg)
		if err != nil {
			return err
		}
		fmt.Printf("Service %s: %s\n", cfg.Service.Name, status)

		// Check tray status
		if isTrayRunning() {
			fmt.Println("System Tray: Running")
		} else {
			fmt.Println("System Tray: Not running")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceStatusCmd)

	serviceCmd.PersistentFlags().StringP("config", "c", "", "config file path")
}

func loadServiceConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	cfg, err := config.LoadOrDefault(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// startTrayApp starts the system tray application
func startTrayApp() error {
	// Check if already running
	if isTrayRunning() {
		return nil // Already running
	}

	// Find baudlink-tray.exe in multiple locations
	trayPath, err := findTrayExecutable()
	if err != nil {
		return err
	}

	cmd := exec.Command(trayPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

// stopTrayApp stops the system tray application
func stopTrayApp() error {
	// Use taskkill to stop the tray application by name
	cmd := exec.Command("taskkill", "/IM", "baudlink-tray.exe", "/F")
	return cmd.Run()
}

// isTrayRunning checks if the tray application is already running
func isTrayRunning() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq baudlink-tray.exe")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}

	return bytes.Contains(out.Bytes(), []byte("baudlink-tray.exe"))
}

// findTrayExecutable attempts to locate baudlink-tray.exe in multiple locations
func findTrayExecutable() (string, error) {
	possiblePaths := []string{
		"baudlink-tray.exe", // current directory
		filepath.Join(filepath.Dir(os.Args[0]), "baudlink-tray.exe"),
		filepath.Join(filepath.Dir(os.Args[0]), "tray", "baudlink-tray.exe"),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("baudlink-tray.exe not found; tried: %v", possiblePaths)
}

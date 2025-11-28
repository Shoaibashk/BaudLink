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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Shoaibashk/BaudLink/config"
	"github.com/Shoaibashk/BaudLink/internal/serial"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for available serial ports",
	Long: `Scan and list all available serial ports on this system.

This command discovers serial ports including USB devices, native ports,
Bluetooth serial ports, and virtual ports.

Example:
  baudlink scan
  baudlink scan --json`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().Bool("json", false, "output in JSON format")
	scanCmd.Flags().BoolP("verbose", "v", false, "show detailed port information")
}

func runScan(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	verbose, _ := cmd.Flags().GetBool("verbose")

	scanner, err := serial.NewScanner(nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	ports, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan ports: %w", err)
	}

	if jsonOutput {
		return printPortsJSON(ports)
	}

	if len(ports) == 0 {
		fmt.Println("No serial ports found.")
		return nil
	}

	fmt.Printf("Found %d serial port(s):\n\n", len(ports))

	for _, port := range ports {
		if verbose {
			printPortVerbose(port)
		} else {
			printPortSimple(port)
		}
	}

	return nil
}

func printPortSimple(port serial.PortInfo) {
	status := ""
	if port.IsOpen {
		status = " [OPEN]"
	}
	fmt.Printf("  %s - %s%s\n", port.Name, port.Description, status)
}

func printPortVerbose(port serial.PortInfo) {
	fmt.Printf("  %s\n", port.Name)
	fmt.Printf("    Description:  %s\n", port.Description)
	fmt.Printf("    Type:         %s\n", port.PortType.String())
	if port.HardwareID != "" {
		fmt.Printf("    Hardware ID:  %s\n", port.HardwareID)
	}
	if port.Manufacturer != "" {
		fmt.Printf("    Manufacturer: %s\n", port.Manufacturer)
	}
	if port.Product != "" {
		fmt.Printf("    Product:      %s\n", port.Product)
	}
	if port.SerialNumber != "" {
		fmt.Printf("    Serial:       %s\n", port.SerialNumber)
	}
	if port.VID != "" && port.PID != "" {
		fmt.Printf("    VID/PID:      %s:%s\n", port.VID, port.PID)
	}
	if port.IsOpen {
		fmt.Printf("    Status:       OPEN (locked by %s)\n", port.LockedBy)
	} else {
		fmt.Printf("    Status:       Available\n")
	}
	fmt.Println()
}

func printPortsJSON(ports []serial.PortInfo) error {
	// Simple JSON output without external dependencies
	fmt.Println("[")
	for i, port := range ports {
		comma := ","
		if i == len(ports)-1 {
			comma = ""
		}
		fmt.Printf(`  {"name": "%s", "description": "%s", "type": "%s", "hardware_id": "%s", "vid": "%s", "pid": "%s", "is_open": %t}%s`+"\n",
			port.Name, port.Description, port.PortType.String(), port.HardwareID, port.VID, port.PID, port.IsOpen, comma)
	}
	fmt.Println("]")
	return nil
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage agent configuration",
	Long: `Manage the BaudLink agent configuration.

Subcommands:
  init    - Create a default configuration file
  show    - Display current configuration
  path    - Show the default configuration file path`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("output")
		if path == "" {
			path = config.DefaultConfigPath()
		}

		cfg := config.DefaultConfig()
		if err := cfg.Save(path); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Configuration file created: %s\n", path)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("config")
		if path == "" {
			path = config.DefaultConfigPath()
		}

		cfg, err := config.LoadOrDefault(path)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Configuration from: %s\n\n", path)
		fmt.Printf("Server:\n")
		fmt.Printf("  gRPC Address:     %s\n", cfg.Server.GRPCAddress)
		fmt.Printf("  Max Connections:  %d\n", cfg.Server.MaxConnections)
		fmt.Printf("  WebSocket:        %v\n", cfg.Server.WebSocketEnabled)
		fmt.Println()
		fmt.Printf("TLS:\n")
		fmt.Printf("  Enabled: %v\n", cfg.TLS.Enabled)
		fmt.Println()
		fmt.Printf("Serial Defaults:\n")
		fmt.Printf("  Baud Rate:        %d\n", cfg.Serial.Defaults.BaudRate)
		fmt.Printf("  Data Bits:        %d\n", cfg.Serial.Defaults.DataBits)
		fmt.Printf("  Stop Bits:        %d\n", cfg.Serial.Defaults.StopBits)
		fmt.Printf("  Scan Interval:    %ds\n", cfg.Serial.ScanInterval)
		fmt.Println()
		fmt.Printf("Logging:\n")
		fmt.Printf("  Level:  %s\n", cfg.Logging.Level)
		fmt.Printf("  Format: %s\n", cfg.Logging.Format)

		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the default configuration file path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.DefaultConfigPath())
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)

	configInitCmd.Flags().StringP("output", "o", "", "output path for config file")
	configShowCmd.Flags().StringP("config", "c", "", "config file path")
}

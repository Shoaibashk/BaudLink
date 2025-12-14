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
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	pb "github.com/Shoaibashk/BaudLink/api/proto"
	"github.com/Shoaibashk/BaudLink/internal/client"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for available serial ports",
	Long: `Scan and list all available serial ports on the connected service.

This command discovers serial ports including USB devices, native ports,
Bluetooth serial ports, and virtual ports.

Example:
  baudlink-cli scan
  baudlink-cli scan --json
  baudlink-cli scan --verbose`,
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

	ctx := context.Background()
	cli := client.NewClient(address)

	// Check connection to service
	if err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to service at %s: %w", address, err)
	}

	// List ports from service
	ports, err := cli.ListPorts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list ports: %w", err)
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

func printPortSimple(p *pb.PortInfo) {
	status := "Available"
	if p.GetIsOpen() {
		status = fmt.Sprintf("Open (locked by: %s)", p.GetLockedBy())
	}
	fmt.Printf("  %s\n    Description: %s\n    Status: %s\n\n",
		p.GetName(), p.GetDescription(), status)
}

func printPortVerbose(p *pb.PortInfo) {
	status := "Available"
	if p.GetIsOpen() {
		status = fmt.Sprintf("Open (locked by: %s)", p.GetLockedBy())
	}
	fmt.Printf("  %s\n", p.GetName())
	fmt.Printf("    Description: %s\n", p.GetDescription())
	fmt.Printf("    Hardware ID: %s\n", p.GetHardwareId())
	fmt.Printf("    Manufacturer: %s\n", p.GetManufacturer())
	fmt.Printf("    Product: %s\n", p.GetProduct())
	fmt.Printf("    Serial Number: %s\n", p.GetSerialNumber())
	fmt.Printf("    Status: %s\n\n", status)
}

func printPortsJSON(ports []*pb.PortInfo) error {
	data, err := json.MarshalIndent(ports, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ports to JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

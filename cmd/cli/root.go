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
	"os"

	"github.com/spf13/cobra"
)

// Version information (will be set by goreleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// Global flags
	address string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "baudlink-cli",
	Short: "BaudLink CLI - Client for the BaudLink Serial Port Service",
	Long: `BaudLink CLI is a command-line client for the BaudLink serial port service.

It communicates with the background BaudLink service via gRPC to manage serial ports.

Features:
  • List available serial ports
  • Open and close serial ports
  • Read and write data to serial ports
  • Configure port settings
  • Monitor port status and statistics

Example usage:
  baudlink-cli scan                          # List available serial ports
  baudlink-cli open COM1 --baud 9600         # Open a port with specific baud rate
  baudlink-cli write COM1 "Hello"            # Write data to an open port
  baudlink-cli read COM1                     # Read data from an open port
  baudlink-cli status COM1                   # Check port status

Connect to a remote service:
  baudlink-cli --address localhost:50051 scan

For more information, visit: https://github.com/Shoaibashk/BaudLink`,
	Version: version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Set version template to include build info
	rootCmd.SetVersionTemplate(`{{.Name}} version {{.Version}}
commit: ` + commit + `
built at: ` + date + `
`)

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&address, "address", "localhost:50051",
		"gRPC service address (can also be set via BAUDLINK_ADDRESS env var)")
}

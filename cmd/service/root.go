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
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "baudlink-service",
	Short: "BaudLink Service - Background Serial Port Manager",
	Long: `BaudLink Service is a background system service for managing serial ports.

It runs silently in the background (system tray on Windows), exposing a
gRPC API on localhost for the BaudLink CLI and other clients to use.

Features:
  • Auto-detect and enumerate serial ports
  • Open, close, and configure serial ports
  • Read and write data with streaming support
  • Port locking (1 client = 1 port)
  • Runs as Windows Service or systemd service
  • System tray application (Windows)

Quick Start:
  baudlink-service serve             # Start the service in foreground
  baudlink-service service install   # Install as system service
  baudlink-service service start     # Start the service
  
Then use the CLI:
  baudlink-cli scan                  # List ports
  baudlink-cli open COM1 --baud 9600 # Open a port

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
}

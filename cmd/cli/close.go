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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Shoaibashk/BaudLink/internal/client"
)

var closeCmd = &cobra.Command{
	Use:   "close <port> <session-id>",
	Short: "Close a serial port",
	Long: `Close an open serial port.

The session ID is returned when the port is opened.

Example:
  baudlink-cli close COM1 "abc123def456"`,
	Args: cobra.ExactArgs(2),
	RunE: runClose,
}

func init() {
	rootCmd.AddCommand(closeCmd)
}

func runClose(cmd *cobra.Command, args []string) error {
	portName := args[0]
	sessionID := args[1]

	ctx := context.Background()
	cli := client.NewClient(address)

	// Check connection to service
	if err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to service at %s: %w", address, err)
	}

	// Close port
	if err := cli.ClosePort(ctx, portName, sessionID); err != nil {
		return fmt.Errorf("failed to close port: %w", err)
	}

	fmt.Printf("Successfully closed port %s\n", portName)
	return nil
}

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
	"time"

	"github.com/spf13/cobra"

	"github.com/Shoaibashk/BaudLink/internal/client"
)

var statusCmd = &cobra.Command{
	Use:   "status <port> <session-id>",
	Short: "Get port status and statistics",
	Long: `Get the current status and statistics of an open port.

Example:
  baudlink-cli status COM1 "abc123def456"`,
	Args: cobra.ExactArgs(2),
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	portName := args[0]
	sessionID := args[1]

	ctx := context.Background()
	cli := client.NewClient(address)

	// Check connection to service
	if err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to service at %s: %w", address, err)
	}

	// Get port status
	status, err := cli.GetPortStatus(ctx, portName, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get port status: %w", err)
	}

	fmt.Printf("Port: %s\n", portName)
	fmt.Printf("Is Open: %v\n", status.GetIsOpen())
	fmt.Printf("Locked By: %s\n", status.GetLockedBy())

	if s := status.GetStatistics(); s != nil {
		fmt.Printf("\nStatistics:\n")
		fmt.Printf("  Bytes Sent: %d\n", s.GetBytesSent())
		fmt.Printf("  Bytes Received: %d\n", s.GetBytesReceived())
		fmt.Printf("  Errors: %d\n", s.GetErrors())
		fmt.Printf("  Last Activity: %s\n", time.Unix(s.GetLastActivity(), 0).UTC())
		fmt.Printf("  Connected Since: %s\n", time.Unix(s.GetOpenedAt(), 0).UTC())
	}

	return nil
}

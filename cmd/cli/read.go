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

var readCmd = &cobra.Command{
	Use:   "read <port> <session-id> [num-bytes]",
	Short: "Read data from a serial port",
	Long: `Read data from an open serial port.

If num-bytes is not specified, it defaults to 1024 bytes.

Example:
  baudlink-cli read COM1 "abc123def456"
  baudlink-cli read COM1 "abc123def456" 256`,
	Args: cobra.RangeArgs(2, 3),
	RunE: runRead,
}

func init() {
	rootCmd.AddCommand(readCmd)
}

func runRead(cmd *cobra.Command, args []string) error {
	portName := args[0]
	sessionID := args[1]

	numBytes := int32(1024)
	if len(args) > 2 {
		var n int
		if _, err := fmt.Sscanf(args[2], "%d", &n); err != nil {
			return fmt.Errorf("invalid number of bytes: %w", err)
		}
		numBytes = int32(n)
	}

	ctx := context.Background()
	cli := client.NewClient(address)

	// Check connection to service
	if err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to service at %s: %w", address, err)
	}

	// Read data
	data, err := cli.Read(ctx, portName, sessionID, numBytes)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	if len(data) == 0 {
		fmt.Println("No data available")
		return nil
	}

	fmt.Printf("Read %d bytes from port %s:\n", len(data), portName)
	fmt.Printf("%s\n", string(data))
	return nil
}

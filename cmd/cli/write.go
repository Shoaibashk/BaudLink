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

var writeCmd = &cobra.Command{
	Use:   "write <port> <session-id> <data>",
	Short: "Write data to a serial port",
	Long: `Write data to an open serial port.

The data can be provided as a string or hex-encoded bytes.

Example:
  baudlink-cli write COM1 "abc123def456" "Hello World"
  baudlink-cli write COM1 "abc123def456" "\\x01\\x02\\x03"`,
	Args: cobra.ExactArgs(3),
	RunE: runWrite,
}

func init() {
	rootCmd.AddCommand(writeCmd)
}

func runWrite(cmd *cobra.Command, args []string) error {
	portName := args[0]
	sessionID := args[1]
	data := []byte(args[2])

	ctx := context.Background()
	cli := client.NewClient(address)

	// Check connection to service
	if err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to service at %s: %w", address, err)
	}

	// Write data
	bytesWritten, err := cli.Write(ctx, portName, sessionID, data)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	fmt.Printf("Successfully wrote %d bytes to port %s\n", bytesWritten, portName)
	return nil
}

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

	pb "github.com/Shoaibashk/BaudLink/api/proto"
	"github.com/Shoaibashk/BaudLink/internal/client"
)

var openCmd = &cobra.Command{
	Use:   "open <port>",
	Short: "Open a serial port",
	Long: `Open a serial port with optional configuration parameters.

Example:
  baudlink-cli open COM1
  baudlink-cli open /dev/ttyUSB0 --baud 9600 --data 8 --stop 1
  baudlink-cli open COM1 --baud 115200 --parity none --flow-control none`,
	Args: cobra.ExactArgs(1),
	RunE: runOpen,
}

var (
	openBaud         int32
	openDataBits     int32
	openStopBits     int32
	openParity       string
	openFlowControl  string
	openReadTimeout  int32
	openWriteTimeout int32
)

func init() {
	rootCmd.AddCommand(openCmd)

	openCmd.Flags().Int32Var(&openBaud, "baud", 9600, "baud rate")
	openCmd.Flags().Int32Var(&openDataBits, "data", 8, "data bits (5-8)")
	openCmd.Flags().Int32Var(&openStopBits, "stop", 1, "stop bits (1, 1.5, or 2)")
	openCmd.Flags().StringVar(&openParity, "parity", "none", "parity (none, odd, even, mark, space)")
	openCmd.Flags().StringVar(&openFlowControl, "flow-control", "none", "flow control (none, hardware, software)")
	openCmd.Flags().Int32Var(&openReadTimeout, "read-timeout", 1000, "read timeout in milliseconds")
	openCmd.Flags().Int32Var(&openWriteTimeout, "write-timeout", 1000, "write timeout in milliseconds")
}

func runOpen(cmd *cobra.Command, args []string) error {
	portName := args[0]

	ctx := context.Background()
	cli := client.NewClient(address)

	// Check connection to service
	if err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to service at %s: %w", address, err)
	}

	// Convert parity string to enum
	parityMap := map[string]pb.Parity{
		"none":  pb.Parity_PARITY_NONE,
		"odd":   pb.Parity_PARITY_ODD,
		"even":  pb.Parity_PARITY_EVEN,
		"mark":  pb.Parity_PARITY_MARK,
		"space": pb.Parity_PARITY_SPACE,
	}
	parity, ok := parityMap[openParity]
	if !ok {
		return fmt.Errorf("invalid parity: %s", openParity)
	}

	// Convert flow control string to enum
	fcMap := map[string]pb.FlowControl{
		"none":     pb.FlowControl_FLOW_CONTROL_NONE,
		"hardware": pb.FlowControl_FLOW_CONTROL_HARDWARE,
		"software": pb.FlowControl_FLOW_CONTROL_SOFTWARE,
	}
	flowControl, ok := fcMap[openFlowControl]
	if !ok {
		return fmt.Errorf("invalid flow control: %s", openFlowControl)
	}

	// Open port
	resp, err := cli.OpenPort(ctx, &pb.OpenPortRequest{
		PortName: portName,
		Config: &pb.PortConfig{
			BaudRate:       uint32(openBaud),
			DataBits:       pb.DataBits(openDataBits),
			StopBits:       pb.StopBits(openStopBits),
			Parity:         parity,
			FlowControl:    flowControl,
			ReadTimeoutMs:  uint32(openReadTimeout),
			WriteTimeoutMs: uint32(openWriteTimeout),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to open port: %w", err)
	}

	fmt.Printf("Successfully opened port %s\n", portName)
	fmt.Printf("Session ID: %s\n", resp.SessionId)
	fmt.Printf("Baud Rate: %d\n", openBaud)
	fmt.Printf("Data Bits: %d\n", openDataBits)
	fmt.Printf("Stop Bits: %d\n", openStopBits)
	fmt.Printf("Parity: %s\n", openParity)
	fmt.Printf("Flow Control: %s\n", openFlowControl)

	return nil
}

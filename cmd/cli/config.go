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

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage port configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <port> <session-id>",
	Short: "Get current port configuration",
	Long: `Get the current configuration of an open port.

Example:
  baudlink-cli config get COM1 "abc123def456"`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <port> <session-id>",
	Short: "Set port configuration",
	Long: `Configure settings for an open port.

Example:
  baudlink-cli config set COM1 "abc123def456" --baud 115200 --data 8`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var (
	cfgBaud         int32
	cfgDataBits     int32
	cfgStopBits     int32
	cfgParity       string
	cfgFlowControl  string
	cfgReadTimeout  int32
	cfgWriteTimeout int32
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)

	configSetCmd.Flags().Int32Var(&cfgBaud, "baud", 0, "baud rate (0 to skip)")
	configSetCmd.Flags().Int32Var(&cfgDataBits, "data", 0, "data bits (0 to skip)")
	configSetCmd.Flags().Int32Var(&cfgStopBits, "stop", 0, "stop bits (0 to skip)")
	configSetCmd.Flags().StringVar(&cfgParity, "parity", "", "parity (none, odd, even, mark, space)")
	configSetCmd.Flags().StringVar(&cfgFlowControl, "flow-control", "", "flow control (none, hardware, software)")
	configSetCmd.Flags().Int32Var(&cfgReadTimeout, "read-timeout", 0, "read timeout in ms (0 to skip)")
	configSetCmd.Flags().Int32Var(&cfgWriteTimeout, "write-timeout", 0, "write timeout in ms (0 to skip)")
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	portName := args[0]
	sessionID := args[1]

	ctx := context.Background()
	cli := client.NewClient(address)

	// Check connection to service
	if err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to service at %s: %w", address, err)
	}

	// Get config
	config, err := cli.GetPortConfig(ctx, portName, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get port config: %w", err)
	}

	fmt.Printf("Configuration for port %s:\n", portName)
	fmt.Printf("  Baud Rate: %d\n", config.BaudRate)
	fmt.Printf("  Data Bits: %d\n", config.DataBits)
	fmt.Printf("  Stop Bits: %d\n", config.StopBits)
	fmt.Printf("  Parity: %s\n", parityToString(config.Parity))
	fmt.Printf("  Flow Control: %s\n", flowControlToString(config.FlowControl))
	fmt.Printf("  Read Timeout: %d ms\n", config.ReadTimeoutMs)
	fmt.Printf("  Write Timeout: %d ms\n", config.WriteTimeoutMs)

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	portName := args[0]
	sessionID := args[1]

	ctx := context.Background()
	cli := client.NewClient(address)

	// Check connection to service
	if err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to service at %s: %w", address, err)
	}

	// Build config update request
	config := &pb.PortConfig{}

	if cfgBaud != 0 {
		config.BaudRate = uint32(cfgBaud)
	}
	if cfgDataBits != 0 {
		config.DataBits = pb.DataBits(cfgDataBits)
	}
	if cfgStopBits != 0 {
		config.StopBits = pb.StopBits(cfgStopBits)
	}
	if cfgParity != "" {
		parityMap := map[string]pb.Parity{
			"none":  pb.Parity_PARITY_NONE,
			"odd":   pb.Parity_PARITY_ODD,
			"even":  pb.Parity_PARITY_EVEN,
			"mark":  pb.Parity_PARITY_MARK,
			"space": pb.Parity_PARITY_SPACE,
		}
		parity, ok := parityMap[cfgParity]
		if !ok {
			return fmt.Errorf("invalid parity: %s", cfgParity)
		}
		config.Parity = parity
	}
	if cfgFlowControl != "" {
		fcMap := map[string]pb.FlowControl{
			"none":     pb.FlowControl_FLOW_CONTROL_NONE,
			"hardware": pb.FlowControl_FLOW_CONTROL_HARDWARE,
			"software": pb.FlowControl_FLOW_CONTROL_SOFTWARE,
		}
		fc, ok := fcMap[cfgFlowControl]
		if !ok {
			return fmt.Errorf("invalid flow control: %s", cfgFlowControl)
		}
		config.FlowControl = fc
	}
	if cfgReadTimeout != 0 {
		config.ReadTimeoutMs = uint32(cfgReadTimeout)
	}
	if cfgWriteTimeout != 0 {
		config.WriteTimeoutMs = uint32(cfgWriteTimeout)
	}

	if err := cli.ConfigurePort(ctx, &pb.ConfigurePortRequest{
		PortName:  portName,
		SessionId: sessionID,
		Config:    config,
	}); err != nil {
		return fmt.Errorf("failed to configure port: %w", err)
	}

	fmt.Printf("Successfully configured port %s\n", portName)
	return nil
}

func parityToString(p pb.Parity) string {
	switch p {
	case pb.Parity_PARITY_NONE:
		return "none"
	case pb.Parity_PARITY_ODD:
		return "odd"
	case pb.Parity_PARITY_EVEN:
		return "even"
	case pb.Parity_PARITY_MARK:
		return "mark"
	case pb.Parity_PARITY_SPACE:
		return "space"
	default:
		return "unknown"
	}
}

func flowControlToString(fc pb.FlowControl) string {
	switch fc {
	case pb.FlowControl_FLOW_CONTROL_NONE:
		return "none"
	case pb.FlowControl_FLOW_CONTROL_HARDWARE:
		return "hardware"
	case pb.FlowControl_FLOW_CONTROL_SOFTWARE:
		return "software"
	default:
		return "unknown"
	}
}

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

	"github.com/Shoaibashk/BaudLink/internal/client"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get service information",
	Long: `Get information about the BaudLink service including version, build info, and capabilities.

Example:
  baudlink-cli info
  baudlink-cli info --json`,
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().Bool("json", false, "output in JSON format")
}

func runInfo(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	ctx := context.Background()
	cli := client.NewClient(address)

	// Check connection to service
	if err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to service at %s: %w", address, err)
	}

	// Get agent info
	info, err := cli.GetAgentInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}

	if jsonOutput {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal info to JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("BaudLink Service Information\n")
	fmt.Printf("=============================\n")
	fmt.Printf("Version: %s\n", info.GetVersion())
	fmt.Printf("Build Commit: %s\n", info.GetBuildCommit())
	fmt.Printf("Build Date: %s\n", info.GetBuildDate())
	fmt.Printf("Operating System: %s\n", info.GetOs())
	fmt.Printf("Architecture: %s\n", info.GetArch())
	fmt.Printf("Uptime (s): %d\n", info.GetUptimeSeconds())

	return nil
}

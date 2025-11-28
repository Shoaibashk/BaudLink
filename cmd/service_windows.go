//go:build windows

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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Shoaibashk/BaudLink/config"
	"github.com/Shoaibashk/BaudLink/service"
)

// serviceCmd represents the service command
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the BaudLink Windows service",
	Long: `Manage the BaudLink agent as a Windows service.

This command allows you to install, uninstall, start, stop, and check the
status of the BaudLink agent running as a Windows service.

Subcommands:
  install   - Install the Windows service
  uninstall - Remove the Windows service
  start     - Start the Windows service
  stop      - Stop the Windows service
  status    - Check the Windows service status`,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Windows service",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServiceConfig(cmd)
		if err != nil {
			return err
		}
		return service.Install(cfg)
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the Windows service",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServiceConfig(cmd)
		if err != nil {
			return err
		}
		return service.Uninstall(cfg)
	},
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Windows service",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServiceConfig(cmd)
		if err != nil {
			return err
		}
		return service.Start(cfg)
	},
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Windows service",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServiceConfig(cmd)
		if err != nil {
			return err
		}
		return service.Stop(cfg)
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the Windows service status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadServiceConfig(cmd)
		if err != nil {
			return err
		}
		status, err := service.Status(cfg)
		if err != nil {
			return err
		}
		fmt.Printf("Service %s: %s\n", cfg.Service.Name, status)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceStatusCmd)

	serviceCmd.PersistentFlags().StringP("config", "c", "", "config file path")
}

func loadServiceConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	cfg, err := config.LoadOrDefault(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

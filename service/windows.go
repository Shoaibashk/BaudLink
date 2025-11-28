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

// Package service provides system service wrappers for BaudLink agent.
package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/Shoaibashk/BaudLink/config"
)

var elog debug.Log

// WindowsService implements the Windows service interface
type WindowsService struct {
	config   *config.Config
	startFn  func() error
	stopFn   func()
}

// NewWindowsService creates a new Windows service
func NewWindowsService(cfg *config.Config, startFn func() error, stopFn func()) *WindowsService {
	return &WindowsService{
		config:  cfg,
		startFn: startFn,
		stopFn:  stopFn,
	}
}

// Execute implements the svc.Handler interface
func (ws *WindowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	// Start the agent
	errChan := make(chan error, 1)
	go func() {
		errChan <- ws.startFn()
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case err := <-errChan:
			if err != nil {
				elog.Error(1, fmt.Sprintf("Agent error: %v", err))
				return false, 1
			}
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				elog.Info(1, "Service stop requested")
				break loop
			default:
				elog.Error(1, fmt.Sprintf("Unexpected control request #%d", c))
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	ws.stopFn()
	return false, 0
}

// Run runs the service
func (ws *WindowsService) Run() error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("failed to determine if running as service: %w", err)
	}

	if isService {
		return ws.runAsService()
	}

	return ws.runInteractive()
}

// runAsService runs as a Windows service
func (ws *WindowsService) runAsService() error {
	var err error
	elog, err = eventlog.Open(ws.config.Service.Name)
	if err != nil {
		return fmt.Errorf("failed to open event log: %w", err)
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("Starting %s service", ws.config.Service.Name))

	err = svc.Run(ws.config.Service.Name, ws)
	if err != nil {
		elog.Error(1, fmt.Sprintf("Service failed: %v", err))
		return err
	}

	elog.Info(1, "Service stopped")
	return nil
}

// runInteractive runs in interactive mode (console)
func (ws *WindowsService) runInteractive() error {
	fmt.Println("Running in interactive mode. Press Ctrl+C to stop.")
	return ws.startFn()
}

// Install installs the Windows service
func Install(cfg *config.Config) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(cfg.Service.Name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", cfg.Service.Name)
	}

	s, err = m.CreateService(cfg.Service.Name, exePath, mgr.Config{
		DisplayName: cfg.Service.DisplayName,
		Description: cfg.Service.Description,
		StartType:   mgr.StartAutomatic,
	}, "serve")
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	// Create event log source
	err = eventlog.InstallAsEventCreate(cfg.Service.Name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("failed to setup event log: %w", err)
	}

	fmt.Printf("Service %s installed successfully\n", cfg.Service.Name)
	return nil
}

// Uninstall removes the Windows service
func Uninstall(cfg *config.Config) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(cfg.Service.Name)
	if err != nil {
		return fmt.Errorf("service %s not found", cfg.Service.Name)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	err = eventlog.Remove(cfg.Service.Name)
	if err != nil {
		fmt.Printf("Warning: failed to remove event log: %v\n", err)
	}

	fmt.Printf("Service %s removed successfully\n", cfg.Service.Name)
	return nil
}

// Start starts the Windows service
func Start(cfg *config.Config) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(cfg.Service.Name)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Printf("Service %s started\n", cfg.Service.Name)
	return nil
}

// Stop stops the Windows service
func Stop(cfg *config.Config) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(cfg.Service.Name)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	timeout := time.Now().Add(30 * time.Second)
	for status.State != svc.Stopped {
		if time.Now().After(timeout) {
			return fmt.Errorf("timeout waiting for service to stop")
		}
		time.Sleep(500 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("failed to query service: %w", err)
		}
	}

	fmt.Printf("Service %s stopped\n", cfg.Service.Name)
	return nil
}

// Status returns the status of the Windows service
func Status(cfg *config.Config) (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(cfg.Service.Name)
	if err != nil {
		return "not installed", nil
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("failed to query service: %w", err)
	}

	switch status.State {
	case svc.Stopped:
		return "stopped", nil
	case svc.StartPending:
		return "starting", nil
	case svc.StopPending:
		return "stopping", nil
	case svc.Running:
		return "running", nil
	case svc.ContinuePending:
		return "continuing", nil
	case svc.PausePending:
		return "pausing", nil
	case svc.Paused:
		return "paused", nil
	default:
		return "unknown", nil
	}
}

// GetConfigPath returns the config path for Windows
func GetConfigPath() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "BaudLink", "agent.yaml")
}

// GetLogPath returns the log path for Windows
func GetLogPath() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "BaudLink", "logs")
}

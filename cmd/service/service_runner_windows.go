//go:build windows

package main

import (
	"fmt"

	"github.com/Shoaibashk/BaudLink/config"
	svc "github.com/Shoaibashk/BaudLink/service"
	winSvc "golang.org/x/sys/windows/svc"
)

// runServiceIfWindows handles Windows service integration. It returns handled=true
// only when the process is actually running as a Windows service. When running
// interactively, it returns handled=false so the caller proceeds with normal
// foreground startup and signal handling.
func runServiceIfWindows(cfg *config.Config, startFn func() error, stopFn func()) (bool, error) {
	isService, err := winSvc.IsWindowsService()
	if err != nil {
		return true, fmt.Errorf("failed to determine if running as service: %w", err)
	}

	if !isService {
		// Not running as a service — let the normal startup path handle it.
		return false, nil
	}

	// Running as a service — use the service wrapper which will block.
	ws := svc.NewWindowsService(cfg, startFn, stopFn)
	return true, ws.Run()
}

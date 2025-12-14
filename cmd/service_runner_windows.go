//go:build windows

package cmd

import (
	"github.com/Shoaibashk/BaudLink/config"
	"github.com/Shoaibashk/BaudLink/service"
)

// runServiceIfWindows handles Windows service integration. It always returns
// handled=true because on Windows we either run as a service or in
// interactive mode via the service wrapper.
func runServiceIfWindows(cfg *config.Config, startFn func() error, stopFn func()) (bool, error) {
	ws := service.NewWindowsService(cfg, startFn, stopFn)
	return true, ws.Run()
}

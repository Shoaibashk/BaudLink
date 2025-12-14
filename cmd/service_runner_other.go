//go:build !windows

package cmd

import "github.com/Shoaibashk/BaudLink/config"

// runServiceIfWindows is a no-op on non-Windows platforms. It returns handled=false
// so the caller continues with normal interactive/foreground startup.
func runServiceIfWindows(cfg *config.Config, startFn func() error, stopFn func()) (bool, error) {
	return false, nil
}

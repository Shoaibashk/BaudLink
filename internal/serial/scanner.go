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

// Package serial provides serial port management and communication functionality.
package serial

import (
	"regexp"
	"runtime"
	"sort"
	"sync"

	"go.bug.st/serial/enumerator"
)

// PortType represents the type of serial port
type PortType int

const (
	PortTypeUnknown PortType = iota
	PortTypeUSB
	PortTypeNative
	PortTypeBluetooth
	PortTypeVirtual
)

// String returns the string representation of PortType
func (p PortType) String() string {
	switch p {
	case PortTypeUSB:
		return "USB"
	case PortTypeNative:
		return "Native"
	case PortTypeBluetooth:
		return "Bluetooth"
	case PortTypeVirtual:
		return "Virtual"
	default:
		return "Unknown"
	}
}

// PortInfo contains information about a serial port
type PortInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	HardwareID   string   `json:"hardware_id"`
	Manufacturer string   `json:"manufacturer"`
	Product      string   `json:"product"`
	SerialNumber string   `json:"serial_number"`
	VID          string   `json:"vid"`
	PID          string   `json:"pid"`
	PortType     PortType `json:"port_type"`
	IsOpen       bool     `json:"is_open"`
	LockedBy     string   `json:"locked_by"`
}

// Scanner handles serial port discovery and enumeration
type Scanner struct {
	mu              sync.RWMutex
	excludePatterns []*regexp.Regexp
	cachedPorts     []PortInfo
	manager         *Manager
}

// NewScanner creates a new port scanner
func NewScanner(excludePatterns []string, manager *Manager) (*Scanner, error) {
	s := &Scanner{
		manager: manager,
	}

	for _, pattern := range excludePatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		s.excludePatterns = append(s.excludePatterns, re)
	}

	return s, nil
}

// Scan discovers all available serial ports
func (s *Scanner) Scan() ([]PortInfo, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}

	var result []PortInfo

	for _, port := range ports {
		// Check if port should be excluded
		if s.isExcluded(port.Name) {
			continue
		}

		info := PortInfo{
			Name:         port.Name,
			Product:      port.Product,
			SerialNumber: port.SerialNumber,
			VID:          port.VID,
			PID:          port.PID,
			PortType:     s.detectPortType(port),
		}

		// Build hardware ID
		if port.VID != "" && port.PID != "" {
			info.HardwareID = "USB\\VID_" + port.VID + "&PID_" + port.PID
		}

		// Set description based on available info
		info.Description = s.buildDescription(port)

		// Check if port is currently open/locked
		if s.manager != nil {
			if session := s.manager.GetSession(port.Name); session != nil {
				info.IsOpen = true
				info.LockedBy = session.ClientID
			}
		}

		result = append(result, info)
	}

	// Sort ports by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	// Cache the results
	s.mu.Lock()
	s.cachedPorts = result
	s.mu.Unlock()

	return result, nil
}

// GetCached returns the last cached port list
func (s *Scanner) GetCached() []PortInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cachedPorts
}

// GetPort returns information about a specific port
func (s *Scanner) GetPort(name string) (*PortInfo, error) {
	ports, err := s.Scan()
	if err != nil {
		return nil, err
	}

	for _, port := range ports {
		if port.Name == name {
			return &port, nil
		}
	}

	return nil, ErrPortNotFound
}

// isExcluded checks if a port should be excluded based on patterns
func (s *Scanner) isExcluded(name string) bool {
	for _, pattern := range s.excludePatterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// detectPortType determines the type of port
func (s *Scanner) detectPortType(port *enumerator.PortDetails) PortType {
	if port.IsUSB {
		return PortTypeUSB
	}

	// Check for Bluetooth ports
	switch runtime.GOOS {
	case "windows":
		// Windows Bluetooth COM ports often have specific names
		if matched, _ := regexp.MatchString(`(?i)bluetooth|bth`, port.Name); matched {
			return PortTypeBluetooth
		}
	case "linux":
		if matched, _ := regexp.MatchString(`/dev/rfcomm`, port.Name); matched {
			return PortTypeBluetooth
		}
	case "darwin":
		if matched, _ := regexp.MatchString(`/dev/.*Bluetooth`, port.Name); matched {
			return PortTypeBluetooth
		}
	}

	// Check for virtual/pseudo terminals
	switch runtime.GOOS {
	case "linux":
		if matched, _ := regexp.MatchString(`/dev/pts/|/dev/pty`, port.Name); matched {
			return PortTypeVirtual
		}
	}

	return PortTypeNative
}

// buildDescription creates a human-readable description for the port
func (s *Scanner) buildDescription(port *enumerator.PortDetails) string {
	if port.Product != "" {
		return port.Product
	}
	if port.IsUSB {
		return "USB Serial Device"
	}
	return "Serial Port"
}

// WatchPorts starts watching for port changes and calls the callback when ports change
func (s *Scanner) WatchPorts(interval int, callback func([]PortInfo)) chan struct{} {
	stop := make(chan struct{})

	if interval <= 0 {
		return stop
	}

	go func() {
		ticker := NewTicker(interval)
		defer ticker.Stop()

		var lastPorts []PortInfo

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				ports, err := s.Scan()
				if err != nil {
					continue
				}

				if !s.portsEqual(lastPorts, ports) {
					lastPorts = ports
					callback(ports)
				}
			}
		}
	}()

	return stop
}

// portsEqual compares two port lists for equality
func (s *Scanner) portsEqual(a, b []PortInfo) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Name != b[i].Name || a[i].IsOpen != b[i].IsOpen {
			return false
		}
	}

	return true
}

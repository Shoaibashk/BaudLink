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
	"context"
	"fmt"
	"log"
	"time"

	"github.com/getlantern/systray"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Shoaibashk/BaudLink/api/proto"
	"github.com/Shoaibashk/BaudLink/config"
	"github.com/Shoaibashk/BaudLink/service"
)

// trayCmd represents the tray command
var trayCmd = &cobra.Command{
	Use:   "tray",
	Short: "Start the BaudLink system tray application",
	Long: `Start the BaudLink system tray application.

This displays a system tray icon that allows you to monitor and control 
the BaudLink service, view available serial ports, and check for port 
lock status from the Windows system tray.

Example:
  baudlink tray
  baudlink tray --config /path/to/config.yaml`,
	RunE: runTray,
}

func init() {
	rootCmd.AddCommand(trayCmd)
	trayCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path")
}

type TrayApp struct {
	cfg       *config.Config
	client    pb.SerialServiceClient
	conn      *grpc.ClientConn
	portsMenu *systray.MenuItem
	portItems []*systray.MenuItem
}

var trayApp *TrayApp

func runTray(cmd *cobra.Command, args []string) error {
	// Load configuration
	var err error
	var cfg *config.Config
	if configFile != "" {
		cfg, err = config.Load(configFile)
	} else {
		cfg, err = config.LoadOrDefault(config.DefaultConfigPath())
	}
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	trayApp = &TrayApp{
		cfg: cfg,
	}

	// Run system tray (blocks until quit)
	systray.Run(trayApp.onReady, trayApp.onExit)
	return nil
}

func (ta *TrayApp) onReady() {
	log.Println("Initializing BaudLink system tray...")

	// Set system tray icon and tooltip
	systray.SetIcon(getIcon())
	systray.SetTitle("BaudLink")
	systray.SetTooltip("BaudLink Serial Agent")

	log.Println("System tray icon should now be visible")

	// Create menu items
	mStatus := systray.AddMenuItem("Status: Checking...", "Service status")
	mStatus.Disable()

	systray.AddSeparator()

	// Create ports submenu
	ta.portsMenu = systray.AddMenuItem("📋 Serial Ports", "Available serial ports")
	mRefresh := systray.AddMenuItem("🔄 Refresh", "Refresh port list and status")

	systray.AddSeparator()

	mStartStop := systray.AddMenuItem("▶️ Start Service", "Start the BaudLink service")

	systray.AddSeparator()

	mAbout := systray.AddMenuItem("ℹ️ About", "About BaudLink")
	mQuit := systray.AddMenuItem("❌ Quit", "Exit system tray")

	// Auto-start service if not running
	go func() {
		log.Println("Checking if service is running...")
		ta.connectToService()
		if ta.client == nil {
			log.Println("Service not running, attempting to start...")
			if err := service.Start(ta.cfg); err != nil {
				log.Printf("Failed to auto-start service: %v", err)
			} else {
				log.Println("Service started successfully")
				time.Sleep(2 * time.Second)
				ta.connectToService()
			}
		}
	}()

	// Initial status update
	time.Sleep(2 * time.Second)
	go ta.updateStatusAndPorts(mStatus, mStartStop)

	// Background goroutine for periodic status and port updates
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ta.updateStatusAndPorts(mStatus, mStartStop)
			}
		}
	}()

	// Handle menu actions
	go func() {
		for {
			select {
			case <-mRefresh.ClickedCh:
				go ta.updateStatusAndPorts(mStatus, mStartStop)
			case <-mStartStop.ClickedCh:
				go ta.toggleService(mStatus, mStartStop)
			case <-mAbout.ClickedCh:
				go ta.showAbout()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (ta *TrayApp) onExit() {
	if ta.conn != nil {
		ta.conn.Close()
	}
	log.Println("BaudLink system tray exited")
}

func (ta *TrayApp) connectToService() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, ta.cfg.Server.GRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("Service not responding: %v", err)
		return
	}

	ta.conn = conn
	ta.client = pb.NewSerialServiceClient(conn)
	log.Println("Connected to BaudLink service")
}

func (ta *TrayApp) updateStatusAndPorts(mStatus *systray.MenuItem, mStartStop *systray.MenuItem) {
	// Try to reconnect if not connected
	if ta.client == nil {
		ta.connectToService()
	}

	// Check if service is actually running
	if ta.client == nil {
		mStatus.SetTitle("Status: ❌ Service Stopped")
		mStartStop.SetTitle("▶️ Start Service")
		mStartStop.Enable()
		systray.SetTooltip("BaudLink: Service Stopped")
		ta.updatePortsMenu(nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	resp, err := ta.client.ListPorts(ctx, &pb.ListPortsRequest{})
	cancel()

	if err == nil {
		mStatus.SetTitle("Status: ✅ Running")
		mStartStop.SetTitle("⏹️ Stop Service")
		mStartStop.Enable()
		systray.SetTooltip("BaudLink: Service Running")
		ta.updatePortsMenu(resp.Ports)
	} else {
		mStatus.SetTitle("Status: ❌ Service Stopped")
		mStartStop.SetTitle("▶️ Start Service")
		mStartStop.Enable()
		systray.SetTooltip("BaudLink: Service Stopped")
		ta.client = nil
		if ta.conn != nil {
			ta.conn.Close()
			ta.conn = nil
		}
		ta.updatePortsMenu(nil)
	}
}

func (ta *TrayApp) updatePortsMenu(ports []*pb.PortInfo) {
	// Clear existing port items
	for _, item := range ta.portItems {
		item.Hide()
	}
	ta.portItems = nil

	if ports == nil || len(ports) == 0 {
		ta.portsMenu.SetTitle("📋 Serial Ports (None)")
		noPortsItem := ta.portsMenu.AddSubMenuItem("No ports available", "")
		noPortsItem.Disable()
		ta.portItems = append(ta.portItems, noPortsItem)
		return
	}

	ta.portsMenu.SetTitle(fmt.Sprintf("📋 Serial Ports (%d)", len(ports)))

	for _, port := range ports {
		statusIcon := "✅"
		tooltip := port.Description
		if port.LockedBy != "" {
			statusIcon = "🔒"
			tooltip = fmt.Sprintf("Locked by %s - %s", port.LockedBy, port.Description)
		}

		title := fmt.Sprintf("%s %s", statusIcon, port.Name)
		portItem := ta.portsMenu.AddSubMenuItem(title, tooltip)
		portItem.Disable()
		ta.portItems = append(ta.portItems, portItem)
	}
}

func (ta *TrayApp) toggleService(mStatus *systray.MenuItem, mStartStop *systray.MenuItem) {
	mStartStop.Disable()
	defer mStartStop.Enable()

	if ta.client == nil {
		// Service is stopped, try to start it
		mStatus.SetTitle("Status: ⏳ Starting...")
		systray.SetTooltip("BaudLink: Starting service...")

		if err := service.Start(ta.cfg); err != nil {
			log.Printf("Failed to start service: %v", err)
			mStatus.SetTitle("Status: ❌ Start Failed")
			systray.SetTooltip("BaudLink: Start Failed")
			return
		}

		// Wait for service to initialize
		time.Sleep(3 * time.Second)
		ta.connectToService()
		ta.updateStatusAndPorts(mStatus, mStartStop)
	} else {
		// Service is running, stop it
		mStatus.SetTitle("Status: ⏳ Stopping...")
		systray.SetTooltip("BaudLink: Stopping service...")

		if err := service.Stop(ta.cfg); err != nil {
			log.Printf("Failed to stop service: %v", err)
			mStatus.SetTitle("Status: ❌ Stop Failed")
			systray.SetTooltip("BaudLink: Stop Failed")
			return
		}

		// Clean up connection
		ta.client = nil
		if ta.conn != nil {
			ta.conn.Close()
			ta.conn = nil
		}

		time.Sleep(1 * time.Second)
		ta.updateStatusAndPorts(mStatus, mStartStop)
	}
}

func (ta *TrayApp) showPorts() {
	if ta.client == nil {
		log.Println("⚠️ Service not connected. Please start the service first.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	resp, err := ta.client.ListPorts(ctx, &pb.ListPortsRequest{})
	cancel()

	if err != nil {
		log.Printf("❌ Failed to list ports: %v", err)
		return
	}

	// Log ports to console (they will appear if running from terminal)
	log.Println("\n╔══════════════════════════════════════════════════════╗")
	log.Println("║         Available Serial Ports                     ║")
	log.Println("╚══════════════════════════════════════════════════════╝")

	if len(resp.Ports) == 0 {
		log.Println("  📭 No serial ports found")
	} else {
		for i, port := range resp.Ports {
			status := "✓ Available"
			statusIcon := "✅"
			if port.LockedBy != "" {
				status = fmt.Sprintf("Locked by %s", port.LockedBy)
				statusIcon = "🔒"
			}

			log.Printf("\n  %s [%d] %s", statusIcon, i+1, port.Name)
			log.Printf("      Description: %s", port.Description)
			if port.Manufacturer != "" {
				log.Printf("      Manufacturer: %s", port.Manufacturer)
			}
			if port.Product != "" {
				log.Printf("      Product: %s", port.Product)
			}
			if port.SerialNumber != "" {
				log.Printf("      Serial Number: %s", port.SerialNumber)
			}
			log.Printf("      Status: %s", status)
		}
	}
	log.Println("\n" + "══════════════════════════════════════════════════════")
}

func (ta *TrayApp) showAbout() {
	log.Println("\n╔══════════════════════════════════════════════════════╗")
	log.Println("║            BaudLink System Tray                    ║")
	log.Println("╚══════════════════════════════════════════════════════╝")
	log.Printf("  Version: %s", version)
	log.Println("  Cross-platform Serial Port Background Service")
	log.Println("  https://github.com/Shoaibashk/BaudLink")
	log.Println("══════════════════════════════════════════════════════")
}

// getIcon returns the icon data for the system tray
// This is a proper 16x16 ICO file that Windows can render correctly
func getIcon() []byte {
	// A complete 16x16 32-bit ICO file (blue circular icon)
	// ICO Header
	icon := []byte{
		0x00, 0x00, // Reserved (must be 0)
		0x01, 0x00, // Type (1 = ICO)
		0x01, 0x00, // Number of images
		// Image Directory Entry
		0x10,       // Width (16 pixels)
		0x10,       // Height (16 pixels)
		0x00,       // Color palette (0 = no palette)
		0x00,       // Reserved
		0x01, 0x00, // Color planes
		0x20, 0x00, // Bits per pixel (32-bit RGBA)
		0x40, 0x04, 0x00, 0x00, // Size of image data (1088 bytes)
		0x16, 0x00, 0x00, 0x00, // Offset to image data (22 bytes)
	}

	// BMP Info Header
	bmpHeader := []byte{
		0x28, 0x00, 0x00, 0x00, // Header size (40 bytes)
		0x10, 0x00, 0x00, 0x00, // Width (16)
		0x20, 0x00, 0x00, 0x00, // Height (32 = 16*2 for icon)
		0x01, 0x00, // Planes
		0x20, 0x00, // Bits per pixel (32)
		0x00, 0x00, 0x00, 0x00, // Compression (0 = none)
		0x00, 0x04, 0x00, 0x00, // Image size (1024 bytes)
		0x00, 0x00, 0x00, 0x00, // X pixels per meter
		0x00, 0x00, 0x00, 0x00, // Y pixels per meter
		0x00, 0x00, 0x00, 0x00, // Colors used
		0x00, 0x00, 0x00, 0x00, // Important colors
	}

	icon = append(icon, bmpHeader...)

	// Create 16x16 pixel data (BGRA format, bottom-up)
	// Simple blue circle on transparent background
	pixels := make([]byte, 16*16*4)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			// Calculate distance from center
			dx := float64(x) - 7.5
			dy := float64(y) - 7.5
			dist := dx*dx + dy*dy

			idx := (y*16 + x) * 4

			if dist < 49 { // Inside circle (radius ~7)
				// Blue color (BGRA)
				pixels[idx+0] = 0xFF // B
				pixels[idx+1] = 0x80 // G
				pixels[idx+2] = 0x00 // R
				pixels[idx+3] = 0xFF // A (opaque)
			} else {
				// Transparent
				pixels[idx+0] = 0x00
				pixels[idx+1] = 0x00
				pixels[idx+2] = 0x00
				pixels[idx+3] = 0x00
			}
		}
	}

	icon = append(icon, pixels...)

	// AND mask (all zeros for proper transparency)
	mask := make([]byte, 16*16/8)
	icon = append(icon, mask...)

	return icon
}

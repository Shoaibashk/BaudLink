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

package main

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
	svc "github.com/Shoaibashk/BaudLink/service"
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
  baudlink-service tray
  baudlink-service tray --config /path/to/config.yaml`,
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
			if err := svc.Start(ta.cfg); err != nil {
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
		status := "✅ Available"
		if port.IsOpen {
			status = "🔒 In Use"
		}

		title := fmt.Sprintf("%s (%s)", port.Name, status)
		item := ta.portsMenu.AddSubMenuItem(title, port.Description)
		item.Disable()
		ta.portItems = append(ta.portItems, item)
	}
}

func (ta *TrayApp) toggleService(mStatus *systray.MenuItem, mStartStop *systray.MenuItem) {
	if ta.client != nil {
		// Attempt to stop service
		if err := svc.Stop(ta.cfg); err != nil {
			log.Printf("Failed to stop service: %v", err)
			return
		}
		ta.client = nil
		if ta.conn != nil {
			ta.conn.Close()
			ta.conn = nil
		}
		mStatus.SetTitle("Status: ❌ Service Stopped")
		mStartStop.SetTitle("▶️ Start Service")
		mStartStop.Enable()
		systray.SetTooltip("BaudLink: Service Stopped")
		return
	}

	// Attempt to start service
	if err := svc.Start(ta.cfg); err != nil {
		log.Printf("Failed to start service: %v", err)
		return
	}

	time.Sleep(2 * time.Second)
	mStatus.SetTitle("Status: ✅ Running")
	mStartStop.SetTitle("⏹️ Stop Service")
	mStartStop.Enable()
	systray.SetTooltip("BaudLink: Service Running")
	ta.connectToService()
}

func (ta *TrayApp) showAbout() {
	log.Println("BaudLink system tray - see https://github.com/Shoaibashk/BaudLink")
}

// getIcon returns embedded icon bytes (placeholder)
func getIcon() []byte {
	// Placeholder icon; in production, embed actual icon bytes.
	return []byte{0x00}
}

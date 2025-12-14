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
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/Shoaibashk/BaudLink/api"
	pb "github.com/Shoaibashk/BaudLink/api/proto"
	"github.com/Shoaibashk/BaudLink/config"
	"github.com/Shoaibashk/BaudLink/internal/serial"
)

var (
	configFile string
	cfg        *config.Config
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the BaudLink service",
	Long: `Start the BaudLink serial port service in the foreground.

The service exposes a gRPC API for managing serial ports, allowing
the CLI and other clients to discover, open, configure, read from, and
write to serial ports on this machine.

On Windows, the service will also display a system tray icon.

Example:
  baudlink-service serve
  baudlink-service serve --config /etc/baudlink/agent.yaml
  baudlink-service serve --address 0.0.0.0:50051`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path")
	serveCmd.Flags().String("address", "", "gRPC server address (overrides config)")
	serveCmd.Flags().Bool("debug", false, "enable debug logging")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Load configuration
	var err error
	if configFile != "" {
		cfg, err = config.Load(configFile)
	} else {
		cfg, err = config.LoadOrDefault(config.DefaultConfigPath())
	}
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply command line overrides
	if addr, _ := cmd.Flags().GetString("address"); addr != "" {
		cfg.Server.GRPCAddress = addr
	}
	if debug, _ := cmd.Flags().GetBool("debug"); debug {
		cfg.Logging.Level = "debug"
	}

	// Setup logging
	setupLogging(cfg)

	log.Printf("Starting BaudLink Service v%s", version)
	log.Printf("gRPC address: %s", cfg.Server.GRPCAddress)
	log.Printf("TLS enabled: %v", cfg.TLS.Enabled)

	// Create serial manager
	serialConfig := serial.PortConfig{
		BaudRate:       cfg.Serial.Defaults.BaudRate,
		DataBits:       cfg.Serial.Defaults.DataBits,
		StopBits:       serial.StopBits(cfg.Serial.Defaults.StopBits),
		Parity:         serial.ParityNone,
		FlowControl:    serial.FlowControlNone,
		ReadTimeoutMs:  cfg.Serial.Defaults.ReadTimeoutMs,
		WriteTimeoutMs: cfg.Serial.Defaults.WriteTimeoutMs,
	}
	manager := serial.NewManager(cfg.Serial.AllowSharedAccess, serialConfig)

	// Create scanner
	scanner, err := serial.NewScanner(cfg.Serial.ExcludePatterns, manager)
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	// Do initial port scan
	ports, err := scanner.Scan()
	if err != nil {
		log.Printf("Warning: initial port scan failed: %v", err)
	} else {
		log.Printf("Found %d serial ports", len(ports))
		for _, port := range ports {
			log.Printf("  - %s (%s)", port.Name, port.Description)
		}
	}

	// Start port watching
	if cfg.Serial.ScanInterval > 0 {
		stopWatch := scanner.WatchPorts(cfg.Serial.ScanInterval, func(ports []serial.PortInfo) {
			log.Printf("Port change detected, %d ports available", len(ports))
		})
		defer close(stopWatch)
	}

	// Create gRPC server options
	var opts []grpc.ServerOption

	// Setup TLS if enabled
	if cfg.TLS.Enabled {
		creds, err := loadTLSCredentials(cfg)
		if err != nil {
			return fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
		log.Println("TLS enabled")
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(opts...)

	// Register services
	serialServer := api.NewSerialServer(manager, scanner, cfg)
	pb.RegisterSerialServiceServer(grpcServer, serialServer)

	// Enable reflection for development/debugging tools like grpcurl
	reflection.Register(grpcServer)

	// Create listener
	listener, err := net.Listen("tcp", cfg.Server.GRPCAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Create server control primitives
	errChan := make(chan error, 1)

	startFn := func() error {
		log.Printf("gRPC server listening on %s", cfg.Server.GRPCAddress)
		if err := grpcServer.Serve(listener); err != nil {
			// Serve returns ErrServerStopped on graceful stop; treat as nil
			errChan <- err
			return err
		}
		return nil
	}

	stopFn := func() {
		log.Println("Shutting down server...")
		grpcServer.GracefulStop()
		manager.CloseAll()
		log.Println("Server stopped")
	}

	// If running on Windows, the platform-specific runner may handle service
	// integration (svc.Run). If it does, it will run the server (foreground
	// or as a service) and return its result.
	if handled, err := runServiceIfWindows(cfg, startFn, stopFn); handled {
		return err
	}

	// Interactive/foreground mode: handle signals and run server normally
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start server in goroutine
	go func() {
		if err := startFn(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		log.Println("Shutdown signal received")
		stopFn()
		// give server a moment to stop
		select {
		case err := <-errChan:
			if err != nil {
				return fmt.Errorf("server error: %w", err)
			}
		case <-time.After(5 * time.Second):
		}
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func loadTLSCredentials(cfg *config.Config) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return credentials.NewTLS(tlsConfig), nil
}

func setupLogging(cfg *config.Config) {
	// Basic logging setup
	// In production, you'd use a more sophisticated logging library
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	if cfg.Logging.File != "" {
		f, err := os.OpenFile(cfg.Logging.File, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Warning: failed to open log file: %v", err)
		} else {
			log.SetOutput(f)
		}
	}
}

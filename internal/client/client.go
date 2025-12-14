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

package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Shoaibashk/BaudLink/api/proto"
)

// Client wraps the gRPC serial service client with connection management
type Client struct {
	addr     string
	conn     *grpc.ClientConn
	grpcConn pb.SerialServiceClient
	mu       sync.RWMutex
}

// NewClient creates a new gRPC client for the given address
func NewClient(addr string) *Client {
	return &Client{
		addr: addr,
	}
}

// Connect establishes a connection to the gRPC service
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // Already connected
	}

	// Create a new connection with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to service at %s: %w", c.addr, err)
	}

	c.conn = conn
	c.grpcConn = pb.NewSerialServiceClient(conn)
	return nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ensureConnection ensures we have an active connection
func (c *Client) ensureConnection(ctx context.Context) error {
	c.mu.RLock()
	if c.conn != nil && c.grpcConn != nil {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	return c.Connect(ctx)
}

// Ping checks if the service is alive
func (c *Client) Ping(ctx context.Context) error {
	if err := c.ensureConnection(ctx); err != nil {
		return err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		c.mu.Lock()
		c.conn = nil
		c.grpcConn = nil
		c.mu.Unlock()
		return fmt.Errorf("service ping failed: %w", err)
	}

	return nil
}

// ListPorts lists all available serial ports
func (c *Client) ListPorts(ctx context.Context) ([]*pb.PortInfo, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.ListPorts(ctx, &pb.ListPortsRequest{})
	if err != nil {
		c.mu.Lock()
		c.conn = nil
		c.grpcConn = nil
		c.mu.Unlock()
		return nil, fmt.Errorf("failed to list ports: %w", err)
	}

	return resp.Ports, nil
}

// GetPortInfo retrieves detailed information about a specific port
func (c *Client) GetPortInfo(ctx context.Context, portName string) (*pb.PortInfo, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.GetPortInfo(ctx, &pb.GetPortInfoRequest{PortName: portName})
	if err != nil {
		return nil, fmt.Errorf("failed to get port info: %w", err)
	}

	return resp, nil
}

// OpenPort opens a serial port with the given configuration
func (c *Client) OpenPort(ctx context.Context, req *pb.OpenPortRequest) (*pb.OpenPortResponse, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.OpenPort(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to open port: %w", err)
	}

	return resp, nil
}

// ClosePort closes an open serial port
func (c *Client) ClosePort(ctx context.Context, portName, sessionID string) error {
	if err := c.ensureConnection(ctx); err != nil {
		return err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := client.ClosePort(ctx, &pb.ClosePortRequest{
		PortName:  portName,
		SessionId: sessionID,
	})
	if err != nil {
		return fmt.Errorf("failed to close port: %w", err)
	}

	return nil
}

// GetPortStatus retrieves the current status of a port
func (c *Client) GetPortStatus(ctx context.Context, portName, sessionID string) (*pb.PortStatus, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.GetPortStatus(ctx, &pb.GetPortStatusRequest{
		PortName: portName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get port status: %w", err)
	}

	return resp, nil
}

// ConfigurePort configures a serial port with the given settings
func (c *Client) ConfigurePort(ctx context.Context, req *pb.ConfigurePortRequest) error {
	if err := c.ensureConnection(ctx); err != nil {
		return err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := client.ConfigurePort(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to configure port: %w", err)
	}

	return nil
}

// GetPortConfig retrieves the current configuration of a port
func (c *Client) GetPortConfig(ctx context.Context, portName, sessionID string) (*pb.PortConfig, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.GetPortConfig(ctx, &pb.GetPortConfigRequest{
		PortName: portName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get port config: %w", err)
	}

	return resp, nil
}

// Write writes data to an open port
func (c *Client) Write(ctx context.Context, portName, sessionID string, data []byte) (int32, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return 0, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.Write(ctx, &pb.WriteRequest{
		PortName:  portName,
		SessionId: sessionID,
		Data:      data,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to write data: %w", err)
	}

	return int32(resp.GetBytesWritten()), nil
}

// Read reads data from an open port
func (c *Client) Read(ctx context.Context, portName, sessionID string, numBytes int32) ([]byte, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.Read(ctx, &pb.ReadRequest{
		PortName:  portName,
		SessionId: sessionID,
		MaxBytes:  uint32(numBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	return resp.Data, nil
}

// StreamRead starts streaming reads from a port
func (c *Client) StreamRead(ctx context.Context, portName, sessionID string) (grpc.ClientStream, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	stream, err := client.StreamRead(ctx, &pb.StreamReadRequest{
		PortName:  portName,
		SessionId: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start stream read: %w", err)
	}

	return stream, nil
}

// StreamWrite starts streaming writes to a port
func (c *Client) StreamWrite(ctx context.Context) (grpc.ClientStream, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	stream, err := client.StreamWrite(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start stream write: %w", err)
	}

	return stream, nil
}

// BiDirectionalStream creates a bidirectional stream with the service
func (c *Client) BiDirectionalStream(ctx context.Context) (grpc.ClientStream, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	stream, err := client.BiDirectionalStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start bidirectional stream: %w", err)
	}

	return stream, nil
}

// GetAgentInfo retrieves information about the service
func (c *Client) GetAgentInfo(ctx context.Context) (*pb.AgentInfo, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	client := c.grpcConn
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.GetAgentInfo(ctx, &pb.GetAgentInfoRequest{})
	if err != nil {
		c.mu.Lock()
		c.conn = nil
		c.grpcConn = nil
		c.mu.Unlock()
		return nil, fmt.Errorf("failed to get agent info: %w", err)
	}

	return resp, nil
}

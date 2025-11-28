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

// Package api provides the gRPC server implementation for BaudLink agent.
package api

import (
	"context"
	"io"
	"runtime"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Shoaibashk/BaudLink/config"
	"github.com/Shoaibashk/BaudLink/internal/serial"

	pb "github.com/Shoaibashk/BaudLink/api/proto"
)

// Version information (set at build time)
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// SerialServer implements the gRPC SerialService
type SerialServer struct {
	pb.UnimplementedSerialServiceServer
	manager   *serial.Manager
	scanner   *serial.Scanner
	config    *config.Config
	startTime time.Time
	readers   map[string]*serial.Reader
}

// NewSerialServer creates a new SerialServer
func NewSerialServer(manager *serial.Manager, scanner *serial.Scanner, cfg *config.Config) *SerialServer {
	return &SerialServer{
		manager:   manager,
		scanner:   scanner,
		config:    cfg,
		startTime: time.Now(),
		readers:   make(map[string]*serial.Reader),
	}
}

// ListPorts returns all available serial ports
func (s *SerialServer) ListPorts(ctx context.Context, req *pb.ListPortsRequest) (*pb.ListPortsResponse, error) {
	ports, err := s.scanner.Scan()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to scan ports: %v", err)
	}

	var response pb.ListPortsResponse
	for _, p := range ports {
		if req.OnlyAvailable && p.IsOpen {
			continue
		}

		response.Ports = append(response.Ports, &pb.PortInfo{
			Name:         p.Name,
			Description:  p.Description,
			HardwareId:   p.HardwareID,
			Manufacturer: p.Manufacturer,
			Product:      p.Product,
			SerialNumber: p.SerialNumber,
			PortType:     convertPortType(p.PortType),
			IsOpen:       p.IsOpen,
			LockedBy:     p.LockedBy,
		})
	}

	return &response, nil
}

// GetPortInfo returns information about a specific port
func (s *SerialServer) GetPortInfo(ctx context.Context, req *pb.GetPortInfoRequest) (*pb.PortInfo, error) {
	if req.PortName == "" {
		return nil, status.Error(codes.InvalidArgument, "port_name is required")
	}

	port, err := s.scanner.GetPort(req.PortName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "port not found: %v", err)
	}

	return &pb.PortInfo{
		Name:         port.Name,
		Description:  port.Description,
		HardwareId:   port.HardwareID,
		Manufacturer: port.Manufacturer,
		Product:      port.Product,
		SerialNumber: port.SerialNumber,
		PortType:     convertPortType(port.PortType),
		IsOpen:       port.IsOpen,
		LockedBy:     port.LockedBy,
	}, nil
}

// OpenPort opens a serial port
func (s *SerialServer) OpenPort(ctx context.Context, req *pb.OpenPortRequest) (*pb.OpenPortResponse, error) {
	if req.PortName == "" {
		return nil, status.Error(codes.InvalidArgument, "port_name is required")
	}

	clientID := req.ClientId
	if clientID == "" {
		clientID = "default-client"
	}

	cfg := s.convertToSerialConfig(req.Config)

	session, err := s.manager.OpenPort(req.PortName, cfg, clientID, req.Exclusive)
	if err != nil {
		if err == serial.ErrPortLocked {
			return &pb.OpenPortResponse{
				Success: false,
				Message: "port is locked by another client",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to open port: %v", err)
	}

	return &pb.OpenPortResponse{
		Success:   true,
		Message:   "port opened successfully",
		SessionId: session.ID,
	}, nil
}

// ClosePort closes a serial port
func (s *SerialServer) ClosePort(ctx context.Context, req *pb.ClosePortRequest) (*pb.ClosePortResponse, error) {
	if req.PortName == "" {
		return nil, status.Error(codes.InvalidArgument, "port_name is required")
	}
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	// Stop any active reader
	if reader, exists := s.readers[req.PortName]; exists {
		reader.Stop()
		delete(s.readers, req.PortName)
	}

	err := s.manager.ClosePort(req.PortName, req.SessionId)
	if err != nil {
		if err == serial.ErrInvalidSession {
			return &pb.ClosePortResponse{
				Success: false,
				Message: "invalid session ID",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to close port: %v", err)
	}

	return &pb.ClosePortResponse{
		Success: true,
		Message: "port closed successfully",
	}, nil
}

// GetPortStatus returns the status of a port
func (s *SerialServer) GetPortStatus(ctx context.Context, req *pb.GetPortStatusRequest) (*pb.PortStatus, error) {
	if req.PortName == "" {
		return nil, status.Error(codes.InvalidArgument, "port_name is required")
	}

	session, err := s.manager.GetStatus(req.PortName)
	if err != nil {
		if err == serial.ErrPortNotOpen {
			return &pb.PortStatus{
				PortName: req.PortName,
				IsOpen:   false,
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to get port status: %v", err)
	}

	return &pb.PortStatus{
		PortName:      session.PortName,
		IsOpen:        true,
		IsLocked:      session.Exclusive,
		LockedBy:      session.ClientID,
		SessionId:     session.ID,
		CurrentConfig: s.convertFromSerialConfig(session.Config),
		Statistics: &pb.PortStatistics{
			BytesSent:     session.Statistics.BytesSent,
			BytesReceived: session.Statistics.BytesReceived,
			Errors:        session.Statistics.Errors,
			OpenedAt:      session.Statistics.OpenedAt.Unix(),
			LastActivity:  session.Statistics.LastActivity.Unix(),
		},
	}, nil
}

// Write writes data to a port
func (s *SerialServer) Write(ctx context.Context, req *pb.WriteRequest) (*pb.WriteResponse, error) {
	if req.PortName == "" {
		return nil, status.Error(codes.InvalidArgument, "port_name is required")
	}
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	n, err := s.manager.Write(req.PortName, req.SessionId, req.Data)
	if err != nil {
		return &pb.WriteResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if req.Flush {
		s.manager.Flush(req.PortName, req.SessionId)
	}

	return &pb.WriteResponse{
		Success:      true,
		BytesWritten: uint32(n),
		Message:      "data written successfully",
	}, nil
}

// Read reads data from a port
func (s *SerialServer) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	if req.PortName == "" {
		return nil, status.Error(codes.InvalidArgument, "port_name is required")
	}
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	maxBytes := int(req.MaxBytes)
	if maxBytes <= 0 {
		maxBytes = 1024
	}

	var data []byte
	var err error

	if req.TimeoutMs > 0 {
		result := serial.ReadWithTimeout(s.manager, req.PortName, req.SessionId, maxBytes, time.Duration(req.TimeoutMs)*time.Millisecond)
		data = result.Data
		err = result.Error
	} else {
		data, err = s.manager.Read(req.PortName, req.SessionId, maxBytes)
	}

	if err != nil {
		return &pb.ReadResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.ReadResponse{
		Success:   true,
		Data:      data,
		BytesRead: uint32(len(data)),
		Message:   "data read successfully",
	}, nil
}

// StreamRead streams data from a port
func (s *SerialServer) StreamRead(req *pb.StreamReadRequest, stream pb.SerialService_StreamReadServer) error {
	if req.PortName == "" {
		return status.Error(codes.InvalidArgument, "port_name is required")
	}
	if req.SessionId == "" {
		return status.Error(codes.InvalidArgument, "session_id is required")
	}

	chunkSize := int(req.ChunkSize)
	if chunkSize <= 0 {
		chunkSize = 1024
	}

	reader := serial.NewReader(s.manager, req.PortName, req.SessionId, chunkSize)
	s.readers[req.PortName] = reader

	if err := reader.Start(stream.Context()); err != nil {
		return status.Errorf(codes.Internal, "failed to start reader: %v", err)
	}
	defer reader.Stop()

	subscription := reader.Subscribe()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case event, ok := <-subscription:
			if !ok {
				return nil
			}

			if event.Error != nil {
				if event.Error == serial.ErrPortClosed {
					return nil
				}
				continue
			}

			chunk := &pb.DataChunk{
				PortName: req.PortName,
				Data:     event.Data,
				Sequence: event.Sequence,
			}

			if req.IncludeTimestamps {
				chunk.Timestamp = event.Timestamp.UnixNano()
			}

			if err := stream.Send(chunk); err != nil {
				return err
			}
		}
	}
}

// StreamWrite writes streaming data to a port
func (s *SerialServer) StreamWrite(stream pb.SerialService_StreamWriteServer) error {
	var totalBytes uint64
	var chunksProcessed uint32

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.StreamWriteResponse{
				Success:           true,
				TotalBytesWritten: totalBytes,
				ChunksProcessed:   chunksProcessed,
				Message:           "stream completed successfully",
			})
		}
		if err != nil {
			return err
		}

		// We need a session ID from somewhere - use first chunk's port
		session := s.manager.GetSession(chunk.PortName)
		if session == nil {
			return status.Error(codes.NotFound, "port not open")
		}

		n, err := s.manager.Write(chunk.PortName, session.ID, chunk.Data)
		if err != nil {
			return status.Errorf(codes.Internal, "write failed: %v", err)
		}

		atomic.AddUint64(&totalBytes, uint64(n))
		atomic.AddUint32(&chunksProcessed, 1)
	}
}

// BiDirectionalStream handles bidirectional streaming
func (s *SerialServer) BiDirectionalStream(stream pb.SerialService_BiDirectionalStreamServer) error {
	ctx := stream.Context()
	
	// Start a goroutine to handle incoming writes
	errChan := make(chan error, 2)

	go func() {
		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				errChan <- nil
				return
			}
			if err != nil {
				errChan <- err
				return
			}

			session := s.manager.GetSession(chunk.PortName)
			if session == nil {
				errChan <- status.Error(codes.NotFound, "port not open")
				return
			}

			_, err = s.manager.Write(chunk.PortName, session.ID, chunk.Data)
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Send back any read data
	// Note: This requires a reader to be set up for the port
	// For now, this is a simplified implementation

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// ConfigurePort configures a port
func (s *SerialServer) ConfigurePort(ctx context.Context, req *pb.ConfigurePortRequest) (*pb.ConfigurePortResponse, error) {
	if req.PortName == "" {
		return nil, status.Error(codes.InvalidArgument, "port_name is required")
	}
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	cfg := s.convertToSerialConfig(req.Config)

	err := s.manager.Configure(req.PortName, req.SessionId, cfg)
	if err != nil {
		return &pb.ConfigurePortResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.ConfigurePortResponse{
		Success: true,
		Message: "port configured successfully",
	}, nil
}

// GetPortConfig returns the current configuration of a port
func (s *SerialServer) GetPortConfig(ctx context.Context, req *pb.GetPortConfigRequest) (*pb.PortConfig, error) {
	if req.PortName == "" {
		return nil, status.Error(codes.InvalidArgument, "port_name is required")
	}

	session, err := s.manager.GetStatus(req.PortName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "port not open: %v", err)
	}

	return s.convertFromSerialConfig(session.Config), nil
}

// Ping checks if the server is alive
func (s *SerialServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	message := req.Message
	if message == "" {
		message = "pong"
	}

	return &pb.PingResponse{
		Message:    message,
		ServerTime: time.Now().Unix(),
	}, nil
}

// GetAgentInfo returns information about the agent
func (s *SerialServer) GetAgentInfo(ctx context.Context, req *pb.GetAgentInfoRequest) (*pb.AgentInfo, error) {
	return &pb.AgentInfo{
		Version:     Version,
		BuildCommit: Commit,
		BuildDate:   BuildDate,
		Os:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		SupportedFeatures: []string{
			"grpc",
			"port-scan",
			"port-lock",
			"streaming",
		},
		Config: &pb.AgentConfig{
			GrpcAddress:    s.config.Server.GRPCAddress,
			TlsEnabled:     s.config.TLS.Enabled,
			MaxConnections: uint32(s.config.Server.MaxConnections),
		},
	}, nil
}

// Helper functions

func (s *SerialServer) convertToSerialConfig(cfg *pb.PortConfig) serial.PortConfig {
	if cfg == nil {
		return serial.PortConfig{
			BaudRate:       s.config.Serial.Defaults.BaudRate,
			DataBits:       s.config.Serial.Defaults.DataBits,
			StopBits:       serial.StopBits(s.config.Serial.Defaults.StopBits),
			Parity:         serial.ParityNone,
			FlowControl:    serial.FlowControlNone,
			ReadTimeoutMs:  s.config.Serial.Defaults.ReadTimeoutMs,
			WriteTimeoutMs: s.config.Serial.Defaults.WriteTimeoutMs,
		}
	}

	return serial.PortConfig{
		BaudRate:       int(cfg.BaudRate),
		DataBits:       int(cfg.DataBits),
		StopBits:       convertStopBits(cfg.StopBits),
		Parity:         convertParity(cfg.Parity),
		FlowControl:    convertFlowControl(cfg.FlowControl),
		ReadTimeoutMs:  int(cfg.ReadTimeoutMs),
		WriteTimeoutMs: int(cfg.WriteTimeoutMs),
	}
}

func (s *SerialServer) convertFromSerialConfig(cfg serial.PortConfig) *pb.PortConfig {
	return &pb.PortConfig{
		BaudRate:       uint32(cfg.BaudRate),
		DataBits:       pb.DataBits(cfg.DataBits),
		StopBits:       convertStopBitsBack(cfg.StopBits),
		Parity:         convertParityBack(cfg.Parity),
		FlowControl:    convertFlowControlBack(cfg.FlowControl),
		ReadTimeoutMs:  uint32(cfg.ReadTimeoutMs),
		WriteTimeoutMs: uint32(cfg.WriteTimeoutMs),
	}
}

func convertPortType(pt serial.PortType) pb.PortType {
	switch pt {
	case serial.PortTypeUSB:
		return pb.PortType_PORT_TYPE_USB
	case serial.PortTypeNative:
		return pb.PortType_PORT_TYPE_NATIVE
	case serial.PortTypeBluetooth:
		return pb.PortType_PORT_TYPE_BLUETOOTH
	case serial.PortTypeVirtual:
		return pb.PortType_PORT_TYPE_VIRTUAL
	default:
		return pb.PortType_PORT_TYPE_UNSPECIFIED
	}
}

func convertStopBits(sb pb.StopBits) serial.StopBits {
	switch sb {
	case pb.StopBits_STOP_BITS_1:
		return serial.StopBits1
	case pb.StopBits_STOP_BITS_1_5:
		return serial.StopBits1Half
	case pb.StopBits_STOP_BITS_2:
		return serial.StopBits2
	default:
		return serial.StopBits1
	}
}

func convertStopBitsBack(sb serial.StopBits) pb.StopBits {
	switch sb {
	case serial.StopBits1:
		return pb.StopBits_STOP_BITS_1
	case serial.StopBits1Half:
		return pb.StopBits_STOP_BITS_1_5
	case serial.StopBits2:
		return pb.StopBits_STOP_BITS_2
	default:
		return pb.StopBits_STOP_BITS_1
	}
}

func convertParity(p pb.Parity) serial.Parity {
	switch p {
	case pb.Parity_PARITY_NONE:
		return serial.ParityNone
	case pb.Parity_PARITY_ODD:
		return serial.ParityOdd
	case pb.Parity_PARITY_EVEN:
		return serial.ParityEven
	case pb.Parity_PARITY_MARK:
		return serial.ParityMark
	case pb.Parity_PARITY_SPACE:
		return serial.ParitySpace
	default:
		return serial.ParityNone
	}
}

func convertParityBack(p serial.Parity) pb.Parity {
	switch p {
	case serial.ParityNone:
		return pb.Parity_PARITY_NONE
	case serial.ParityOdd:
		return pb.Parity_PARITY_ODD
	case serial.ParityEven:
		return pb.Parity_PARITY_EVEN
	case serial.ParityMark:
		return pb.Parity_PARITY_MARK
	case serial.ParitySpace:
		return pb.Parity_PARITY_SPACE
	default:
		return pb.Parity_PARITY_NONE
	}
}

func convertFlowControl(fc pb.FlowControl) serial.FlowControl {
	switch fc {
	case pb.FlowControl_FLOW_CONTROL_NONE:
		return serial.FlowControlNone
	case pb.FlowControl_FLOW_CONTROL_HARDWARE:
		return serial.FlowControlHardware
	case pb.FlowControl_FLOW_CONTROL_SOFTWARE:
		return serial.FlowControlSoftware
	default:
		return serial.FlowControlNone
	}
}

func convertFlowControlBack(fc serial.FlowControl) pb.FlowControl {
	switch fc {
	case serial.FlowControlNone:
		return pb.FlowControl_FLOW_CONTROL_NONE
	case serial.FlowControlHardware:
		return pb.FlowControl_FLOW_CONTROL_HARDWARE
	case serial.FlowControlSoftware:
		return pb.FlowControl_FLOW_CONTROL_SOFTWARE
	default:
		return pb.FlowControl_FLOW_CONTROL_NONE
	}
}

/*
BaudLink gRPC Test Client

Tests the BaudLink gRPC server by:
  1. Connecting to the agent
  2. Listing available serial ports
  3. Opening a port (optional)
  4. Writing data (optional)
  5. Reading data with streaming
  6. Closing the port

Usage:
  grpcclient -addr localhost:50051 -port COM3 -baud 115200 -write "AT\r\n" -read-time 10
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Shoaibashk/BaudLink/api/proto"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "BaudLink gRPC server address")
	portName := flag.String("port", "", "Serial port to open (e.g., COM3). Leave empty to just list ports.")
	baud := flag.Uint("baud", 9600, "Baud rate")
	writeData := flag.String("write", "", "Data to write after opening the port")
	readTimeSec := flag.Int("read-time", 5, "Seconds to read data from the port")
	flag.Parse()

	fmt.Println("╔════════════════════════════════════════════╗")
	fmt.Println("║       BaudLink gRPC Test Client            ║")
	fmt.Println("╚════════════════════════════════════════════╝")
	fmt.Printf("Server: %s\n\n", *addr)

	// Connect to BaudLink gRPC server
	conn, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("❌ Failed to connect to BaudLink: %v\n   Make sure 'baudlink serve' is running.", err)
	}
	defer conn.Close()
	fmt.Println("✅ Connected to BaudLink")

	// Create the client using the generated protobuf client
	client := pb.NewSerialServiceClient(conn)

	// 1. Ping
	fmt.Println("━━━ Ping ━━━")
	pingResp, err := client.Ping(context.Background(), &pb.PingRequest{Message: "hello"})
	if err != nil {
		log.Printf("⚠ Ping failed: %v", err)
	} else {
		fmt.Printf("Pong: %s (server time: %d)\n", pingResp.Message, pingResp.ServerTime)
	}
	fmt.Println()

	// 2. Get Agent Info
	fmt.Println("━━━ Agent Info ━━━")
	info, err := client.GetAgentInfo(context.Background(), &pb.GetAgentInfoRequest{})
	if err != nil {
		log.Printf("⚠ GetAgentInfo failed: %v", err)
	} else {
		fmt.Printf("Version:  %s\n", info.Version)
		fmt.Printf("OS/Arch:  %s/%s\n", info.Os, info.Arch)
		fmt.Printf("Uptime:   %d seconds\n", info.UptimeSeconds)
		fmt.Printf("Features: %v\n", info.SupportedFeatures)
	}
	fmt.Println()

	// 3. List Ports
	fmt.Println("━━━ Serial Ports ━━━")
	listResp, err := client.ListPorts(context.Background(), &pb.ListPortsRequest{})
	if err != nil {
		log.Fatalf("❌ ListPorts failed: %v", err)
	}
	if len(listResp.Ports) == 0 {
		fmt.Println("No serial ports found")
	} else {
		for _, p := range listResp.Ports {
			status := "available"
			if p.IsOpen {
				status = fmt.Sprintf("open (locked by %s)", p.LockedBy)
			}
			fmt.Printf("  • %s - %s [%s]\n", p.Name, p.Description, status)
		}
	}
	fmt.Println()

	// If no port specified, stop here
	if *portName == "" {
		fmt.Println("💡 Tip: Use -port COM3 to open and test a specific port")
		return
	}

	// 4. Open Port
	fmt.Printf("━━━ Open Port: %s @ %d baud ━━━\n", *portName, *baud)
	openResp, err := client.OpenPort(context.Background(), &pb.OpenPortRequest{
		PortName: *portName,
		Config: &pb.PortConfig{
			BaudRate:      uint32(*baud),
			DataBits:      pb.DataBits_DATA_BITS_8,
			StopBits:      pb.StopBits_STOP_BITS_1,
			Parity:        pb.Parity_PARITY_NONE,
			FlowControl:   pb.FlowControl_FLOW_CONTROL_NONE,
			ReadTimeoutMs: 1000,
		},
		ClientId:  "grpc-test-client",
		Exclusive: true,
	})
	if err != nil {
		log.Fatalf("❌ OpenPort failed: %v", err)
	}
	if !openResp.Success {
		log.Fatalf("❌ OpenPort error: %s", openResp.Message)
	}
	sessionID := openResp.SessionId
	fmt.Printf("✅ Port opened (session: %s)\n\n", sessionID)

	// Ensure we close the port on exit
	defer func() {
		fmt.Println("\n━━━ Closing Port ━━━")
		closeResp, err := client.ClosePort(context.Background(), &pb.ClosePortRequest{
			PortName:  *portName,
			SessionId: sessionID,
		})
		if err != nil {
			log.Printf("⚠ ClosePort failed: %v", err)
		} else if closeResp.Success {
			fmt.Println("✅ Port closed")
		} else {
			fmt.Printf("⚠ ClosePort: %s\n", closeResp.Message)
		}
	}()

	// 5. Write Data (optional)
	if *writeData != "" {
		fmt.Println("━━━ Write Data ━━━")
		writeResp, err := client.Write(context.Background(), &pb.WriteRequest{
			PortName:  *portName,
			SessionId: sessionID,
			Data:      []byte(*writeData),
			Flush:     true,
		})
		if err != nil {
			log.Printf("⚠ Write failed: %v", err)
		} else if writeResp.Success {
			fmt.Printf("✅ Wrote %d bytes: %q\n", writeResp.BytesWritten, *writeData)
		} else {
			fmt.Printf("⚠ Write: %s\n", writeResp.Message)
		}
		fmt.Println()
	}

	// 6. Read Data (streaming)
	fmt.Printf("━━━ Reading Data (for %d seconds, press Ctrl+C to stop) ━━━\n", *readTimeSec)
	readCtx, readCancel := context.WithTimeout(context.Background(), time.Duration(*readTimeSec)*time.Second)
	defer readCancel()

	stream, err := client.StreamRead(readCtx, &pb.StreamReadRequest{
		PortName:          *portName,
		SessionId:         sessionID,
		ChunkSize:         256,
		IncludeTimestamps: true,
	})
	if err != nil {
		log.Printf("⚠ StreamRead failed: %v", err)
		return
	}

	bytesTotal := 0
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if readCtx.Err() != nil {
				fmt.Printf("\n⏱ Read timeout (%d seconds)\n", *readTimeSec)
			} else {
				log.Printf("⚠ StreamRead error: %v", err)
			}
			break
		}
		if len(chunk.Data) > 0 {
			bytesTotal += len(chunk.Data)
			fmt.Printf("← %s", string(chunk.Data))
		}
	}
	if bytesTotal > 0 {
		fmt.Printf("\n\n📊 Total received: %d bytes\n", bytesTotal)
	} else {
		fmt.Println("(no data received)")
	}
}

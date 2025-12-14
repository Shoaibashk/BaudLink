package integration

import (
	"context"
	"encoding/json"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Shoaibashk/BaudLink/internal/client"
)

func freePort(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()
	addr := l.Addr().String()
	return addr
}

func TestServiceBasic(t *testing.T) {
	addr := freePort(t)
	t.Logf("Using address %s", addr)

	// build service binary into temp dir
	out := filepath.Join(t.TempDir(), "baudlink-service-test")
	// On Windows, ensure .exe suffix
	if runtime.GOOS == "windows" {
		out = out + ".exe"
	}
	cmdBuild := exec.Command("go", "build", "-o", out, "../cmd/service")
	if outb, err := cmdBuild.CombinedOutput(); err != nil {
		t.Fatalf("failed to build service: %v output: %s", err, string(outb))
	}

	cmd := exec.Command(out, "serve", "--address", addr)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				t.Logf("service stdout: %s", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				t.Logf("service stderr: %s", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		cmd.Wait()
	}()

	// wait for readiness
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var c *client.Client
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("service didn't become ready: %v", ctx.Err())
		default:
			cc := client.NewClient(addr)
			if err := cc.Connect(context.Background()); err == nil {
				t.Logf("connected to service")
				c = cc
				break
			} else {
				t.Logf("connect attempt failed: %v", err)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	defer c.Close()

	// Check agent info
	info, err := c.GetAgentInfo(context.Background())
	if err != nil {
		t.Fatalf("GetAgentInfo failed: %v", err)
	}
	if info == nil {
		t.Fatalf("GetAgentInfo returned nil")
	}
	// ensure config.grpc_address matches
	cfg := info.GetConfig()
	if cfg == nil {
		t.Fatalf("info.Config is nil")
	}
	if cfg.GetGrpcAddress() != addr {
		t.Fatalf("grpc_address mismatch: expected %s got %s", addr, cfg.GetGrpcAddress())
	}

	// Ensure we can list ports (should succeed but may be empty)
	ports, err := c.ListPorts(context.Background())
	if err != nil {
		t.Fatalf("ListPorts failed: %v", err)
	}
	// ports can be empty, but ensure JSON marshalling works for them
	if _, err := json.Marshal(ports); err != nil {
		t.Fatalf("failed to marshal ports: %v", err)
	}
}

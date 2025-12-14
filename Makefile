# BaudLink Makefile
# Cross-platform serial port background service

.PHONY: all build clean test lint proto install uninstall help

# Variables
BUILD_DIR=build
BINARY_NAME=$(BUILD_DIR)/baudlink
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/Shoaibashk/BaudLink/cmd.version=$(VERSION) -X github.com/Shoaibashk/BaudLink/cmd.commit=$(COMMIT) -X github.com/Shoaibashk/BaudLink/cmd.date=$(DATE)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Default target
all: build

# Platform specific helpers
ifeq ($(OS),Windows_NT)
MKDIR_P = powershell -NoProfile -Command New-Item -ItemType Directory -Force -Path
RM = powershell -NoProfile -Command Remove-Item -Recurse -Force
else
MKDIR_P = mkdir -p
RM = rm -rf
endif

# Build the binary
build:
	@$(MKDIR_P) $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .
ifeq ($(OS),Windows_NT)
	@echo "Building tray application..."
	$(GOBUILD) $(LDFLAGS) -ldflags "-H windowsgui" -o $(BUILD_DIR)/baudlink-tray.exe .
endif

# Build for all platforms
build-all: build-linux build-windows build-darwin build-arm

build-linux:
	@$(MKDIR_P) $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/baudlink_linux_amd64 .

build-windows:
	@$(MKDIR_P) $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/baudlink_windows_amd64.exe .

build-darwin:
	@$(MKDIR_P) $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/baudlink_darwin_amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/baudlink_darwin_arm64 .

build-arm:
	@$(MKDIR_P) $(BUILD_DIR)
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/baudlink_linux_arm7 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/baudlink_linux_arm64 .

# Clean build artifacts
clean:
	$(GOCLEAN)
	$(RM) $(BUILD_DIR)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	$(GOCMD) vet ./...
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Generate protobuf files
proto:
	@if command -v protoc > /dev/null; then \
		protoc --go_out=. --go_opt=paths=source_relative \
			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
			api/proto/serial.proto; \
	else \
		echo "protoc not installed. Install with: https://grpc.io/docs/protoc-installation/"; \
	fi

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Install the binary
install: build
	$(GOCMD) install $(LDFLAGS) .

# Uninstall the binary
uninstall:
	$(RM) $(shell $(GOCMD) env GOPATH)/bin/baudlink

# Run the server
run: build
	$(BINARY_NAME) serve

# Run with debug logging
run-debug: build
	$(BINARY_NAME) serve --debug

# Scan for ports
scan: build
	$(BINARY_NAME) scan -v

# Show version
version: build
	$(BINARY_NAME) version

# Build and run tray application (Windows only)
build-tray:
ifeq ($(OS),Windows_NT)
	@$(MKDIR_P) $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -ldflags "-H windowsgui" -o $(BUILD_DIR)/baudlink-tray.exe .
else
	@echo "Tray application is only available on Windows"
endif

tray: build-tray
ifeq ($(OS),Windows_NT)
	$(BUILD_DIR)/baudlink-tray.exe tray
else
	@echo "Tray application is only available on Windows"
endif

# Install development tools
dev-tools:
	$(GOCMD) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GOCMD) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Help
help:
	@echo "BaudLink - Cross-platform Serial Port Background Service"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build        Build the binary for current platform"
	@echo "  build-all    Build for all supported platforms"
	@echo "  clean        Remove build artifacts"
	@echo "  test         Run tests"
	@echo "  lint         Run linter"
	@echo "  proto        Generate protobuf files"
	@echo "  deps         Download and tidy dependencies"
	@echo "  install      Install the binary"
	@echo "  uninstall    Uninstall the binary"
	@echo "  run          Build and run the server"
	@echo "  run-debug    Build and run with debug logging"
	@echo "  scan         Scan for serial ports"
	@echo "  build-tray   Build system tray GUI application (Windows only)"
	@echo "  tray         Build and run system tray application (Windows only)"
	@echo "  dev-tools    Install development tools"
	@echo "  help         Show this help"

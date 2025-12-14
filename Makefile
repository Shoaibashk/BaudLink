# BaudLink Makefile
# Cross-platform serial port background service

.PHONY: all build clean test lint proto install uninstall help

# Variables
BUILD_DIR=build
EXE=
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# LDFLAGS for each binary (inject version info into respective packages)
LDFLAGS_CLI=-ldflags "-X github.com/Shoaibashk/BaudLink/cmd/cli.version=$(VERSION) -X github.com/Shoaibashk/BaudLink/cmd/cli.commit=$(COMMIT) -X github.com/Shoaibashk/BaudLink/cmd/cli.date=$(DATE)"
LDFLAGS_SERVICE=-ldflags "-X github.com/Shoaibashk/BaudLink/cmd/service.version=$(VERSION) -X github.com/Shoaibashk/BaudLink/cmd/service.commit=$(COMMIT) -X github.com/Shoaibashk/BaudLink/cmd/service.date=$(DATE)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
# Platform specific helpers

ifeq ($(OS),Windows_NT)
EXE=.exe
MKDIR_P = powershell -NoProfile -Command New-Item -ItemType Directory -Force -Path
RM = powershell -NoProfile -Command Remove-Item -Recurse -Force
else
MKDIR_P = mkdir -p
RM = rm -rf
endif

# Binary paths (include EXE suffix on Windows)
SERVICE_BINARY=$(BUILD_DIR)/baudlink-service$(EXE)
CLI_BINARY=$(BUILD_DIR)/baudlink-cli$(EXE)
build: build-service build-cli

build-service:
	@$(MKDIR_P) $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS_SERVICE) -o $(SERVICE_BINARY) ./cmd/service

# Build the Windows GUI variant of the service (use manually on Windows)
build-service-windows:
	@$(MKDIR_P) $(BUILD_DIR)
	@echo "Building Windows GUI service executable..."
	$(GOBUILD) $(LDFLAGS_SERVICE) -ldflags "-H windowsgui" -o $(SERVICE_BINARY) ./cmd/service

build-cli:
	@$(MKDIR_P) $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS_CLI) -o $(CLI_BINARY) ./cmd/cli

# Packaging
PACKAGE_DIR=$(BUILD_DIR)/packages

package-linux: build-service
	@$(MKDIR_P) $(PACKAGE_DIR)
	@echo "Creating Linux package..."
	bash packaging/linux/package-linux.sh
	@echo "Linux package created in $(PACKAGE_DIR)"

package-windows: build-service build-cli
	@$(MKDIR_P) $(PACKAGE_DIR)
	@echo "Creating Windows installer (requires makensis)"
	@pwsh -NoProfile -File packaging/windows/package-windows.ps1 -OutDir $(PACKAGE_DIR)
	@echo "Windows installer created in $(PACKAGE_DIR)"

package: package-linux package-windows
	@echo "All packages created in $(PACKAGE_DIR)"

install: build-service build-cli
	# Install both service and cli into GOPATH/bin
	$(GOCMD) install $(LDFLAGS_SERVICE) ./cmd/service
	$(GOCMD) install $(LDFLAGS_CLI) ./cmd/cli

run: build-service
	$(SERVICE_BINARY) serve
run-debug: build-service
	$(SERVICE_BINARY) serve --debug
scan: build-cli
	$(CLI_BINARY) scan -v

# Clean build artifacts
clean:
	$(GOCLEAN)
	$(RM) $(BUILD_DIR)

# Run tests with vet and optional linter
vet-lint:
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

# Uninstall binaries (remove from GOPATH/bin)
uninstall:
	$(RM) $(shell $(GOCMD) env GOPATH)/bin/baudlink-service || true
	$(RM) $(shell $(GOCMD) env GOPATH)/bin/baudlink-cli || true

# CLI run helpers (use built binaries)
service-run: build-service
	$(SERVICE_BINARY) serve

service-run-debug: build-service
	$(SERVICE_BINARY) serve --debug

cli-scan: build-cli
	$(CLI_BINARY) scan -v

# Build tray (Windows GUI service)
build-tray: build-service
ifeq ($(OS),Windows_NT)
	@echo "Windows tray is part of the service binary (built with -H windowsgui)"
	@echo "To build the GUI service executable run: make build-service on Windows"
else
	@echo "Tray application (GUI) is only available on Windows"
endif

tray: build-service
ifeq ($(OS),Windows_NT)
	$(SERVICE_BINARY) tray
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
	@echo "  package-linux Create a Linux tarball (systemd unit + installer)"
	@echo "  package-windows Create a Windows NSIS installer (requires makensis)"

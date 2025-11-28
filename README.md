# BaudLink

A command-line interface tool for serial communication.

## Features

- Cross-platform support (Linux, macOS, Windows)
- Built with Go and Cobra CLI framework
- Automated releases with GoReleaser

## Installation

### From Releases

Download the latest release from the [Releases page](https://github.com/Shoaibashk/BaudLink/releases).

### From Source

```bash
go install github.com/Shoaibashk/BaudLink@latest
```

## Usage

```bash
# Show help
baudlink --help

# Show version
baudlink version
```

## Development

### Prerequisites

- Go 1.22 or later (use latest stable version)

### Building

```bash
# Build the binary
go build -o baudlink .

# Run tests
go test -v ./...

# Run linter
go vet ./...
```

### Project Structure

```
.
├── cmd/           # CLI commands
│   ├── root.go    # Root command
│   └── version.go # Version command
├── internal/      # Internal packages (not importable)
├── pkg/           # Reusable packages
├── main.go        # Application entry point
├── go.mod         # Go module definition
└── .goreleaser.yaml # GoReleaser configuration
```

## Release

Releases are automated using GoReleaser. To create a new release:

1. Tag a new version: `git tag v0.1.0`
2. Push the tag: `git push origin v0.1.0`
3. GitHub Actions will automatically build and publish the release

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
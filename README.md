# indiekku

![Tests](https://github.com/MironCo/indiekku/actions/workflows/test.yml/badge.svg)

**indiekku** is a lightweight game server orchestration tool for Unity multiplayer games. It manages Docker containers running Unity dedicated servers, providing a simple CLI and REST API for server lifecycle management.

## Features

- **Simple CLI** - Start, stop, and list game servers with intuitive commands
- **Docker-based** - Isolated server instances with automatic port management
- **REST API** - Programmatic server control via HTTP endpoints
- **Auto-discovery** - Automatically detects Unity server binaries
- **Background daemon** - API server runs in the background for persistent management
- **Port allocation** - Automatic port assignment starting from 7777

## Quick Start

### Prerequisites

- Go 1.23.2 or later
- Docker installed and running

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/indiekku.git
cd indiekku

# Build the binary
make build

# The binary will be in bin/indiekku
```

### Basic Usage

```bash
# Start the API server (runs in background)
./bin/indiekku serve

# Start a game server (auto-assigns port 7777, 7778, etc.)
./bin/indiekku start

# Start on a specific port
./bin/indiekku start --port 7779

# List running servers
./bin/indiekku ps

# Stop a server
./bin/indiekku stop unity-server-7777

# View logs
./bin/indiekku logs

# Shutdown the API server
./bin/indiekku shutdown
```

## Project Structure

```
indiekku/
├── cmd/indiekku/          # CLI entry point
├── internal/
│   ├── api/               # REST API handlers
│   ├── client/            # HTTP client for CLI
│   ├── docker/            # Docker container management
│   ├── server/            # Server binary detection
│   └── state/             # In-memory state management
├── game_server/           # Place your Unity server build here
├── Dockerfile             # Container image for Unity servers
└── Makefile              # Build automation
```

## How It Works

1. **Place your Unity server build** in the `game_server/` directory
2. **indiekku automatically detects** executables (`.x86_64` or `.exe`)
3. **Docker image is built** with Unity dependencies on first start
4. **Each server runs** in an isolated container with host networking
5. **State is tracked** in-memory with thread-safe operations

## API Endpoints

The API server runs on `localhost:8080` by default.

### Health Check
```bash
GET /health
```

### Start Server
```bash
POST /api/v1/servers/start
Content-Type: application/json

{
  "port": "7777"  // optional
}
```

### List Servers
```bash
GET /api/v1/servers
```

### Stop Server
```bash
DELETE /api/v1/servers/:container_name
```

## Development

```bash
# Run tests
make test

# Run with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Clean build artifacts
make clean

# Tidy dependencies
make tidy
```

## Architecture

indiekku uses a **client-server architecture**:

- **API Server** (`indiekku serve`) - Long-running daemon that manages Docker containers and maintains state
- **CLI Commands** - HTTP clients that communicate with the API server

This design allows the API to run continuously while CLI commands execute and exit cleanly.

## Configuration

Currently configured via constants:
- **API Port**: `8080`
- **Base game port**: `7777`
- **Container prefix**: `unity-server-`
- **Docker image**: `unity-server`
- **Server directory**: `game_server/`

## Roadmap

- [ ] Configuration file support
- [ ] Player count tracking via heartbeat
- [ ] Automatic server restart on crash
- [ ] Multiple server build support
- [ ] Metrics and monitoring
- [ ] Persistent state (Redis/SQLite)
- [ ] Web dashboard

## Version

**v0.1.0** - Initial release

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

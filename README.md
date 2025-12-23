# indiekku

![Tests](https://github.com/MironCo/indiekku/actions/workflows/test.yml/badge.svg)
[![Known Vulnerabilities](https://snyk.io/test/github/mironco/indiekku/badge.svg)](https://snyk.io/test/github/mironco/indiekku)

**indiekku** is a lightweight game server orchestration tool for Unity multiplayer games. It manages Docker containers running Unity dedicated servers, providing a simple CLI, REST API, and web UI for server lifecycle management.

## Features

- **Simple CLI** - Start, stop, and list game servers with intuitive commands
- **Web UI** - Modern web interface for uploading and managing server builds
- **History Tracking** - SQLite-powered persistent history of server events and uploads
- **API Key Authentication** - Secure API access with automatically generated keys
- **Rolling Deployment Support** - Upload new builds without affecting running servers
- **Docker-based** - Isolated server instances with automatic port management
- **REST API** - Programmatic server control via HTTP endpoints
- **Auto-discovery** - Automatically detects Unity server binaries
- **Background daemon** - API server runs in the background for persistent management
- **Port allocation** - Automatic port assignment starting from 7777

## Quick Start

### Prerequisites

- Docker installed and running

### Installation

```bash
# Install indiekku
curl -fsSL https://indiekku.mironsulicz.dev/install.sh | bash
```

Or download the latest release manually from the [releases page](https://github.com/MironCo/indiekku/releases).

### Basic Usage

#### 1. Start the Server

```bash
# Start the API server (runs in background)
indiekku serve
```

On first run, indiekku will generate an API key and display it. **Save this key** - you'll need it to authenticate.

```
======================================================================
  NEW API KEY GENERATED
======================================================================

  Your API Key: a1b2c3d4e5f6...

  This key has been saved to: .indiekku_apikey
  Keep this key secure - you'll need it to authenticate API requests.
======================================================================
```

#### 2. Access the Web UI

Navigate to `http://localhost:8080` and login with your API key.

**Upload a Server Build:**
- Drag and drop a ZIP file containing your Unity server build, or
- Click "Browse Files" to select a file
- The build will be automatically extracted and the Docker image rebuilt

**Start Servers:**
- Click "Start Server" to launch a new game server instance
- Optionally specify a port, or leave empty for auto-assignment

**Manage Servers:**
- View all running servers with names (e.g., "legendary-sword"), ports, player counts, and uptime
- Stop servers with one click
- Server list auto-refreshes every 5 seconds

**View History:**
- Navigate to the History page to view server events and upload history
- Server Events: Track all server start/stop events with duration
- Upload History: View all build uploads with success/failure status
- History auto-refreshes every 10 seconds

#### 3. CLI Commands

```bash
# Start a game server (auto-assigns port 7777, 7778, etc.)
indiekku start

# Start on a specific port
indiekku start --port 7779

# List running servers
indiekku ps

# Stop a server (use server name from 'ps' output)
indiekku stop legendary-sword

# View logs
indiekku logs

# Shutdown the API server
indiekku shutdown
```

## Project Structure

```
indiekku/
├── cmd/indiekku/          # CLI entry point
├── internal/
│   ├── api/               # REST API handlers
│   ├── client/            # HTTP client for CLI
│   ├── docker/            # Docker container management
│   ├── history/           # SQLite history tracking
│   ├── server/            # Server binary detection
│   └── state/             # In-memory state management
├── web/                   # Web UI source files
├── game_server/           # Place your Unity server build here
├── Dockerfile             # Container image for Unity servers
└── Makefile              # Build automation
```

## How It Works

1. **Place your Unity server build** in the `game_server/` directory (or upload via Web UI)
2. **indiekku automatically detects** executables (`.x86_64` or `.exe`)
3. **Docker image is built** with Unity dependencies on first start
4. **Each server runs** in an isolated container with host networking
5. **State is tracked** in-memory with thread-safe operations
6. **History is persisted** to SQLite database (`indiekku.db`) for server events and uploads

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

### Get Server History
```bash
GET /api/v1/history/servers
GET /api/v1/history/servers?container=legendary-sword  # Filter by container
```

### Get Upload History
```bash
GET /api/v1/history/uploads
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
- **Container naming**: Random video game themed names (e.g., "legendary-sword", "crimson-dragon")
- **Docker image**: `unity-server`
- **Server directory**: `game_server/`

## Roadmap

- [x] Web dashboard
- [x] Persistent history tracking (SQLite)
- [ ] Configuration file support
- [ ] Player count tracking via heartbeat
- [ ] Automatic server restart on crash
- [ ] Multiple server build support
- [ ] Metrics and monitoring

## Version

**Current Version**: v0.5.0 - History tracking and persistence

See [CHANGELOG.md](CHANGELOG.md) for full version history and release notes.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

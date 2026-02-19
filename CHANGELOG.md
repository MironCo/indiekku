# Changelog

All notable changes to this project will be documented in this file.

## [v0.7.0] - 2025

### Added
- **Self-Signed TLS for Management GUI** - GUI port (9090) now served over HTTPS with a persisted self-signed certificate; browser warning only shown once
- **Unity SDK Client** - `IndiekkuClient.cs` with coroutine-based `FindMatch()`, `JoinServer()`, `ListServers()`, and `Ping()` methods for game clients
- **Unity SDK Server Improvements** - `PlayerJoined()`, `PlayerLeft()`, `PlayerCount`, and `IsFull` helpers for thread-safe player tracking
- **Unity Preprocessor Guards** - `#if UNITY_DEDICATED_SERVER || UNITY_SERVER` guards on server SDK, negation guards on client SDK

### Changed
- Removed global `max_players` config from matchmaking UI; max players now handled per-server by the Unity SDK

## [v0.6.0] - 2025

### Added
- **Matchmaking API** - REST endpoints for server registration, player matchmaking, and JWT join tokens
- **Match Web UI** - Dedicated matchmaking page showing live server list, configuration, and endpoint copy helper
- **Match Proxy** - Built-in reverse proxy to the matchmaking server for CORS-free frontend access

### Changed
- Matchmaking configuration (public IP, port, token secret) surfaced in the Web UI

## [v0.5.0] - 2024

### Added
- **SQLite History Tracking** - Persistent history database for server events and upload records
- **History Web UI** - Dedicated history page with server events and upload history in side-by-side view
- **Server Event Tracking** - Track all server start/stop events with uptime duration
- **Upload History** - Record all build uploads with success/failure status and file metadata
- **Brutalist UI Design** - Clean, functional interface with responsive grid layouts
- **Comprehensive Tests** - Full test coverage for history database operations

### Changed
- Database files (*.db) now ignored in git

## [v0.4.0] - 2024

### Added
- **Embedded Dockerfile** - Dockerfile is now embedded in the binary for true single-binary distribution
- **Automatic Docker image rebuild** - Uploading a new server build now automatically rebuilds the Docker image
- **Start Server button in Web UI** - Launch new game server instances directly from the web interface
- Updated install script for streamlined deployment

### Changed
- Archive files (.zip, .tar.gz) now ignored in git
- Improved .gitignore for cleaner repository

## [v0.3.0] - 2024

### Fixed
- Fixed critical file upload permission errors
- Fixed recursive binary detection for subdirectories in ZIP uploads
- Fixed Docker container execute permissions for server binaries

### Added
- Enhanced web UI with drag-and-drop file upload
- Added video game themed random server name generation (e.g., "legendary-sword", "crimson-dragon")
- Improved server list table with auto-refresh and overflow handling
- Full-width responsive UI layout

### Changed
- Shutdown command now stops all running containers before API shutdown

## [v0.2.0] - 2024

### Added
- Web UI for build management
- API key authentication
- File upload support for server builds
- Rolling deployment capability

## [v0.1.0] - 2024

### Added
- Initial release
- Basic CLI commands (start, stop, ps, logs, shutdown)
- Docker-based server management
- REST API
- Auto-discovery of Unity server binaries
- Background daemon mode

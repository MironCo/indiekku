# indiekku API Reference

The API server runs on `http://localhost:3000` by default.

---

## Authentication

All `/api/v1/` endpoints require a Bearer token in the `Authorization` header.

```
Authorization: Bearer <your-api-key>
```

Your API key is generated on first run and saved to `.indiekku_apikey`.

### CSRF Tokens

State-changing endpoints (`POST`, `DELETE`) also require a CSRF token. Fetch one first, then include it in subsequent requests.

```
X-CSRF-Token: <csrf-token>
```

**Endpoints that require CSRF:** `POST /servers/start`, `DELETE /servers/:name`, `POST /heartbeat`, `POST /upload`, `POST /dockerfiles/active`

---

## Endpoints

### Health

#### `GET /health`

No authentication required.

**Response `200`**
```json
{ "status": "ok" }
```

---

### CSRF

#### `GET /api/v1/csrf-token`

Returns a CSRF token to use with state-changing requests.

**Response `200`**
```json
{ "csrf_token": "abc123..." }
```

---

### Servers

#### `POST /api/v1/servers/start`

Spawns a new game server container. If no port is specified, the next available port starting from `7777` is assigned automatically. If no command is specified, indiekku will try to auto-detect a binary in `game_server/`, falling back to the Dockerfile's `CMD`.

**Headers**
```
Authorization: Bearer <api-key>
X-CSRF-Token: <csrf-token>
Content-Type: application/json
```

**Request body** (all fields optional)
```json
{
  "port": "7777",
  "command": "/app/MyServer.x86_64",
  "args": ["-port", "7777", "-batchmode"]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `port` | string | External port to expose. Auto-assigned if omitted. |
| `command` | string | Overrides the auto-detected binary and Dockerfile CMD. |
| `args` | string[] | Arguments passed to the command. Defaults to `["-port", "<port>"]` when a binary is auto-detected. |

**Response `201`**
```json
{
  "container_name": "legendary-sword",
  "port": "7777",
  "message": "Container started successfully"
}
```

**Response `409`** — port already in use
```json
{ "error": "Port 7777 is already in use" }
```

---

#### `GET /api/v1/servers`

Lists all running servers.

**Headers**
```
Authorization: Bearer <api-key>
```

**Response `200`**
```json
{
  "servers": [
    {
      "container_name": "legendary-sword",
      "port": "7777",
      "command": "/app/MyServer.x86_64",
      "args": ["-port", "7777"],
      "player_count": 3,
      "started_at": "2025-02-14T15:49:00Z"
    }
  ],
  "count": 1
}
```

---

#### `GET /api/v1/servers/:name`

Gets a single running server by container name.

**Headers**
```
Authorization: Bearer <api-key>
```

**Response `200`**
```json
{
  "container_name": "legendary-sword",
  "port": "7777",
  "command": "/app/MyServer.x86_64",
  "args": ["-port", "7777"],
  "player_count": 3,
  "started_at": "2025-02-14T15:49:00Z"
}
```

**Response `404`** — server not found
```json
{ "error": "Server not found: legendary-sword" }
```

---

#### `DELETE /api/v1/servers/:name`

Stops and removes a running server container.

**Headers**
```
Authorization: Bearer <api-key>
X-CSRF-Token: <csrf-token>
```

**Response `200`**
```json
{ "message": "Server legendary-sword stopped successfully" }
```

**Response `404`** — server not found
```json
{ "error": "Server not found: legendary-sword" }
```

---

#### `GET /api/v1/servers/:name/logs`

Returns the last 5 minutes of stdout/stderr from a container.

**Headers**
```
Authorization: Bearer <api-key>
```

**Response `200`**
```json
{
  "container_name": "legendary-sword",
  "logs": "Server started on port 7777\nPlayer connected..."
}
```

---

#### `POST /api/v1/heartbeat`

Updates the player count for a running server. Call this periodically from your game server to keep the dashboard count accurate.

**Headers**
```
Authorization: Bearer <api-key>
X-CSRF-Token: <csrf-token>
Content-Type: application/json
```

**Request body**
```json
{
  "container_name": "legendary-sword",
  "player_count": 5
}
```

**Response `200`**
```json
{ "message": "Heartbeat received" }
```

**Response `404`** — server not found
```json
{ "error": "Server not found: legendary-sword" }
```

---

### Uploads

#### `POST /api/v1/upload`

Uploads a new server build (ZIP archive). The ZIP is extracted into `game_server/`, and the Docker image is rebuilt automatically.

**Headers**
```
Authorization: Bearer <api-key>
X-CSRF-Token: <csrf-token>
Content-Type: multipart/form-data
```

**Form fields**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `server_build` | file | Yes | ZIP archive containing your server binary and assets. Max 500 MB. |
| `preset` | string | No | Dockerfile preset to use (`unity` or `binary`). Ignored if `dockerfile` is provided. |
| `dockerfile` | file | No | Custom Dockerfile. Overrides `preset`. |
| `default_port` | string | No | Internal port your server listens on (e.g. `7777`). |

**Response `200`**
```json
{
  "message": "Release uploaded successfully",
  "file": "server.zip",
  "size": 123456
}
```

---

### Dockerfiles

#### `GET /api/v1/dockerfiles/presets`

Lists all built-in Dockerfile presets with their content.

**Headers**
```
Authorization: Bearer <api-key>
```

**Response `200`**
```json
{
  "presets": [
    { "name": "unity", "content": "FROM debian:13-slim\n..." },
    { "name": "binary", "content": "FROM debian:13-slim\n..." }
  ]
}
```

---

#### `GET /api/v1/dockerfiles/active`

Returns the currently active Dockerfile used for new container builds.

**Headers**
```
Authorization: Bearer <api-key>
```

**Response `200`**
```json
{
  "name": "unity",
  "content": "FROM debian:13-slim\n..."
}
```

---

#### `POST /api/v1/dockerfiles/active`

Sets the active Dockerfile. Accepts either a preset name (JSON) or a custom Dockerfile file (multipart). Removes the existing Docker image so it's rebuilt on the next server start.

**Option 1 — Select a preset**

```
Authorization: Bearer <api-key>
X-CSRF-Token: <csrf-token>
Content-Type: application/json
```
```json
{ "preset": "unity" }
```

**Option 2 — Upload a custom Dockerfile**

```
Authorization: Bearer <api-key>
X-CSRF-Token: <csrf-token>
Content-Type: multipart/form-data
```

| Field | Type | Description |
|-------|------|-------------|
| `dockerfile` | file | A valid Dockerfile (must contain a `FROM` instruction). |

**Response `200`**
```json
{
  "message": "Active Dockerfile set to preset: unity",
  "name": "unity"
}
```

---

#### `GET /api/v1/dockerfiles/history`

Returns the last 100 Dockerfile changes.

**Headers**
```
Authorization: Bearer <api-key>
```

**Response `200`**
```json
{
  "history": [
    {
      "id": 1,
      "name": "unity",
      "source": "preset:unity",
      "timestamp": "2025-02-14T15:49:00Z",
      "notes": "Set via API"
    }
  ],
  "count": 1
}
```

---

### History

#### `GET /api/v1/history/servers`

Returns the last 100 server start/stop events. Filter by container with the `container_name` query param.

**Headers**
```
Authorization: Bearer <api-key>
```

**Query params**

| Param | Description |
|-------|-------------|
| `container_name` | Filter events to a specific container. |

**Example**
```
GET /api/v1/history/servers?container_name=legendary-sword
```

**Response `200`**
```json
{
  "events": [
    {
      "id": 1,
      "container_name": "legendary-sword",
      "event_type": "start",
      "port": "7777",
      "timestamp": "2025-02-14T15:49:00Z",
      "duration": 0
    },
    {
      "id": 2,
      "container_name": "legendary-sword",
      "event_type": "stop",
      "port": "7777",
      "timestamp": "2025-02-14T15:55:00Z",
      "duration": 360
    }
  ],
  "count": 2
}
```

---

#### `GET /api/v1/history/uploads`

Returns the last 100 upload events.

**Headers**
```
Authorization: Bearer <api-key>
```

**Response `200`**
```json
{
  "uploads": [
    {
      "id": 1,
      "filename": "server.zip",
      "file_size": 123456,
      "timestamp": "2025-02-14T15:49:00Z",
      "success": true,
      "notes": "Upload and Docker rebuild successful"
    }
  ],
  "count": 1
}
```

---

## Quick Reference

| Method | Endpoint | Auth | CSRF | Description |
|--------|----------|------|------|-------------|
| `GET` | `/health` | — | — | Health check |
| `GET` | `/api/v1/csrf-token` | ✓ | — | Get CSRF token |
| `POST` | `/api/v1/servers/start` | ✓ | ✓ | Start a server |
| `GET` | `/api/v1/servers` | ✓ | — | List servers |
| `GET` | `/api/v1/servers/:name` | ✓ | — | Get server |
| `DELETE` | `/api/v1/servers/:name` | ✓ | ✓ | Stop a server |
| `GET` | `/api/v1/servers/:name/logs` | ✓ | — | Get container logs |
| `POST` | `/api/v1/heartbeat` | ✓ | ✓ | Update player count |
| `POST` | `/api/v1/upload` | ✓ | ✓ | Upload server build |
| `GET` | `/api/v1/dockerfiles/presets` | ✓ | — | List Dockerfile presets |
| `GET` | `/api/v1/dockerfiles/active` | ✓ | — | Get active Dockerfile |
| `POST` | `/api/v1/dockerfiles/active` | ✓ | ✓ | Set active Dockerfile |
| `GET` | `/api/v1/dockerfiles/history` | ✓ | — | Dockerfile change history |
| `GET` | `/api/v1/history/servers` | ✓ | — | Server event history |
| `GET` | `/api/v1/history/uploads` | ✓ | — | Upload history |

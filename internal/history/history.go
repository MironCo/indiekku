package history

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	EventTypeStart = "start"
	EventTypeStop  = "stop"
)

// HistoryManager handles database operations for tracking history
type HistoryManager struct {
	db *sql.DB
}

// ServerEvent represents a server start/stop event
type ServerEvent struct {
	ID            int       `json:"id"`
	ContainerName string    `json:"container_name"`
	EventType     string    `json:"event_type"`
	Port          string    `json:"port"`
	Timestamp     time.Time `json:"timestamp"`
	Duration      *int64    `json:"duration,omitempty"` // Duration in seconds for stopped servers
}

// UploadHistory represents an upload event
type UploadHistory struct {
	ID        int       `json:"id"`
	Filename  string    `json:"filename"`
	FileSize  int64     `json:"file_size"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Notes     string    `json:"notes,omitempty"`
}

// NewHistoryManager creates a new history manager and initializes the database
func NewHistoryManager(dbPath string) (*HistoryManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	manager := &HistoryManager{db: db}

	// Initialize tables
	if err := manager.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return manager, nil
}

// initTables creates the necessary tables if they don't exist
func (h *HistoryManager) initTables() error {
	// Create server_events table
	serverEventsTable := `
	CREATE TABLE IF NOT EXISTS server_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		container_name TEXT NOT NULL,
		event_type TEXT NOT NULL,
		port TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		duration INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_server_events_timestamp ON server_events(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_server_events_container ON server_events(container_name);
	`

	if _, err := h.db.Exec(serverEventsTable); err != nil {
		return fmt.Errorf("failed to create server_events table: %w", err)
	}

	// Create upload_history table
	uploadHistoryTable := `
	CREATE TABLE IF NOT EXISTS upload_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		timestamp DATETIME NOT NULL,
		success INTEGER NOT NULL,
		notes TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_upload_history_timestamp ON upload_history(timestamp DESC);
	`

	if _, err := h.db.Exec(uploadHistoryTable); err != nil {
		return fmt.Errorf("failed to create upload_history table: %w", err)
	}

	return nil
}

// RecordServerStart records a server start event
func (h *HistoryManager) RecordServerStart(containerName, port string) error {
	query := `INSERT INTO server_events (container_name, event_type, port, timestamp) VALUES (?, ?, ?, ?)`
	_, err := h.db.Exec(query, containerName, EventTypeStart, port, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record server start: %w", err)
	}
	return nil
}

// RecordServerStop records a server stop event
func (h *HistoryManager) RecordServerStop(containerName, port string, startTime time.Time) error {
	duration := int64(time.Since(startTime).Seconds())
	query := `INSERT INTO server_events (container_name, event_type, port, timestamp, duration) VALUES (?, ?, ?, ?, ?)`
	_, err := h.db.Exec(query, containerName, EventTypeStop, port, time.Now(), duration)
	if err != nil {
		return fmt.Errorf("failed to record server stop: %w", err)
	}
	return nil
}

// RecordUpload records an upload event
func (h *HistoryManager) RecordUpload(filename string, fileSize int64, success bool, notes string) error {
	query := `INSERT INTO upload_history (filename, file_size, timestamp, success, notes) VALUES (?, ?, ?, ?, ?)`
	_, err := h.db.Exec(query, filename, fileSize, time.Now(), success, notes)
	if err != nil {
		return fmt.Errorf("failed to record upload: %w", err)
	}
	return nil
}

// GetServerEvents retrieves server events, optionally filtered by container name
func (h *HistoryManager) GetServerEvents(containerName string, limit int) ([]ServerEvent, error) {
	var query string
	var args []interface{}

	if containerName != "" {
		query = `SELECT id, container_name, event_type, port, timestamp, duration FROM server_events WHERE container_name = ? ORDER BY timestamp DESC LIMIT ?`
		args = []interface{}{containerName, limit}
	} else {
		query = `SELECT id, container_name, event_type, port, timestamp, duration FROM server_events ORDER BY timestamp DESC LIMIT ?`
		args = []interface{}{limit}
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query server events: %w", err)
	}
	defer rows.Close()

	var events []ServerEvent
	for rows.Next() {
		var event ServerEvent
		var duration sql.NullInt64
		if err := rows.Scan(&event.ID, &event.ContainerName, &event.EventType, &event.Port, &event.Timestamp, &duration); err != nil {
			return nil, fmt.Errorf("failed to scan server event: %w", err)
		}
		if duration.Valid {
			event.Duration = &duration.Int64
		}
		events = append(events, event)
	}

	return events, nil
}

// GetUploadHistory retrieves upload history
func (h *HistoryManager) GetUploadHistory(limit int) ([]UploadHistory, error) {
	query := `SELECT id, filename, file_size, timestamp, success, notes FROM upload_history ORDER BY timestamp DESC LIMIT ?`
	rows, err := h.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query upload history: %w", err)
	}
	defer rows.Close()

	var uploads []UploadHistory
	for rows.Next() {
		var upload UploadHistory
		var notes sql.NullString
		if err := rows.Scan(&upload.ID, &upload.Filename, &upload.FileSize, &upload.Timestamp, &upload.Success, &notes); err != nil {
			return nil, fmt.Errorf("failed to scan upload history: %w", err)
		}
		if notes.Valid {
			upload.Notes = notes.String
		}
		uploads = append(uploads, upload)
	}

	return uploads, nil
}

// Close closes the database connection
func (h *HistoryManager) Close() error {
	return h.db.Close()
}

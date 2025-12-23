package history

import (
	"os"
	"testing"
	"time"
)

func TestNewHistoryManager(t *testing.T) {
	// Create a temporary database file
	dbPath := "test_history.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	if manager.db == nil {
		t.Error("expected non-nil database connection")
	}
}

func TestRecordServerStart(t *testing.T) {
	dbPath := "test_server_start.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	err = manager.RecordServerStart("test-container", "7777")
	if err != nil {
		t.Fatalf("failed to record server start: %v", err)
	}

	// Verify the event was recorded
	events, err := manager.GetServerEvents("test-container", 10)
	if err != nil {
		t.Fatalf("failed to get server events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.ContainerName != "test-container" {
		t.Errorf("got container name %q, want %q", event.ContainerName, "test-container")
	}
	if event.EventType != EventTypeStart {
		t.Errorf("got event type %q, want %q", event.EventType, EventTypeStart)
	}
	if event.Port != "7777" {
		t.Errorf("got port %q, want %q", event.Port, "7777")
	}
	if event.Duration != nil {
		t.Error("start event should not have duration")
	}
}

func TestRecordServerStop(t *testing.T) {
	dbPath := "test_server_stop.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	startTime := time.Now().Add(-1 * time.Hour)
	err = manager.RecordServerStop("test-container", "7777", startTime)
	if err != nil {
		t.Fatalf("failed to record server stop: %v", err)
	}

	// Verify the event was recorded
	events, err := manager.GetServerEvents("test-container", 10)
	if err != nil {
		t.Fatalf("failed to get server events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.EventType != EventTypeStop {
		t.Errorf("got event type %q, want %q", event.EventType, EventTypeStop)
	}
	if event.Duration == nil {
		t.Fatal("stop event should have duration")
	}
	// Duration should be approximately 1 hour (3600 seconds)
	if *event.Duration < 3590 || *event.Duration > 3610 {
		t.Errorf("got duration %d seconds, want approximately 3600", *event.Duration)
	}
}

func TestGetServerEvents_FilteredByContainer(t *testing.T) {
	dbPath := "test_filtered_events.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Record events for different containers
	manager.RecordServerStart("container-1", "7777")
	manager.RecordServerStart("container-2", "7778")
	manager.RecordServerStart("container-1", "7777")

	// Get events for container-1 only
	events, err := manager.GetServerEvents("container-1", 10)
	if err != nil {
		t.Fatalf("failed to get server events: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("got %d events for container-1, want 2", len(events))
	}

	// Verify all events are for container-1
	for _, event := range events {
		if event.ContainerName != "container-1" {
			t.Errorf("got container name %q, want %q", event.ContainerName, "container-1")
		}
	}
}

func TestGetServerEvents_AllContainers(t *testing.T) {
	dbPath := "test_all_events.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Record events for different containers
	manager.RecordServerStart("container-1", "7777")
	manager.RecordServerStart("container-2", "7778")
	manager.RecordServerStart("container-3", "7779")

	// Get all events (empty container name)
	events, err := manager.GetServerEvents("", 10)
	if err != nil {
		t.Fatalf("failed to get server events: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("got %d events, want 3", len(events))
	}
}

func TestGetServerEvents_Limit(t *testing.T) {
	dbPath := "test_events_limit.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Record 5 events
	for i := 0; i < 5; i++ {
		manager.RecordServerStart("test-container", "7777")
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// Get only 3 events
	events, err := manager.GetServerEvents("", 3)
	if err != nil {
		t.Fatalf("failed to get server events: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("got %d events with limit 3, want 3", len(events))
	}
}

func TestRecordUpload_Success(t *testing.T) {
	dbPath := "test_upload_success.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	err = manager.RecordUpload("server.zip", 1024000, true, "Upload successful")
	if err != nil {
		t.Fatalf("failed to record upload: %v", err)
	}

	// Verify the upload was recorded
	uploads, err := manager.GetUploadHistory(10)
	if err != nil {
		t.Fatalf("failed to get upload history: %v", err)
	}

	if len(uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(uploads))
	}

	upload := uploads[0]
	if upload.Filename != "server.zip" {
		t.Errorf("got filename %q, want %q", upload.Filename, "server.zip")
	}
	if upload.FileSize != 1024000 {
		t.Errorf("got file size %d, want %d", upload.FileSize, 1024000)
	}
	if !upload.Success {
		t.Error("expected success to be true")
	}
	if upload.Notes != "Upload successful" {
		t.Errorf("got notes %q, want %q", upload.Notes, "Upload successful")
	}
}

func TestRecordUpload_Failure(t *testing.T) {
	dbPath := "test_upload_failure.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	err = manager.RecordUpload("bad-file.zip", 0, false, "Invalid file format")
	if err != nil {
		t.Fatalf("failed to record upload: %v", err)
	}

	// Verify the upload was recorded
	uploads, err := manager.GetUploadHistory(10)
	if err != nil {
		t.Fatalf("failed to get upload history: %v", err)
	}

	if len(uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(uploads))
	}

	upload := uploads[0]
	if upload.Success {
		t.Error("expected success to be false")
	}
	if upload.Notes != "Invalid file format" {
		t.Errorf("got notes %q, want %q", upload.Notes, "Invalid file format")
	}
}

func TestGetUploadHistory_Limit(t *testing.T) {
	dbPath := "test_upload_limit.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Record 5 uploads
	for i := 0; i < 5; i++ {
		manager.RecordUpload("file.zip", 1024, true, "Test upload")
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// Get only 2 uploads
	uploads, err := manager.GetUploadHistory(2)
	if err != nil {
		t.Fatalf("failed to get upload history: %v", err)
	}

	if len(uploads) != 2 {
		t.Errorf("got %d uploads with limit 2, want 2", len(uploads))
	}
}

func TestGetUploadHistory_OrderedByTimestamp(t *testing.T) {
	dbPath := "test_upload_order.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Record uploads with slight delays
	manager.RecordUpload("first.zip", 1024, true, "First")
	time.Sleep(10 * time.Millisecond)
	manager.RecordUpload("second.zip", 2048, true, "Second")
	time.Sleep(10 * time.Millisecond)
	manager.RecordUpload("third.zip", 3072, true, "Third")

	uploads, err := manager.GetUploadHistory(10)
	if err != nil {
		t.Fatalf("failed to get upload history: %v", err)
	}

	// Should be ordered newest first
	if len(uploads) < 3 {
		t.Fatalf("expected at least 3 uploads, got %d", len(uploads))
	}

	if uploads[0].Filename != "third.zip" {
		t.Errorf("expected newest upload first, got %q", uploads[0].Filename)
	}
	if uploads[2].Filename != "first.zip" {
		t.Errorf("expected oldest upload last, got %q", uploads[2].Filename)
	}
}

func TestServerLifecycle(t *testing.T) {
	dbPath := "test_lifecycle.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Simulate server lifecycle
	startTime := time.Now()
	manager.RecordServerStart("game-server", "7777")

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	manager.RecordServerStop("game-server", "7777", startTime)

	// Get events
	events, err := manager.GetServerEvents("game-server", 10)
	if err != nil {
		t.Fatalf("failed to get server events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events (start and stop), got %d", len(events))
	}

	// First event should be stop (newest first)
	if events[0].EventType != EventTypeStop {
		t.Errorf("expected first event to be stop, got %q", events[0].EventType)
	}
	if events[0].Duration == nil {
		t.Error("stop event should have duration")
	}

	// Second event should be start
	if events[1].EventType != EventTypeStart {
		t.Errorf("expected second event to be start, got %q", events[1].EventType)
	}
}

func TestClose(t *testing.T) {
	dbPath := "test_close.db"
	defer os.Remove(dbPath)

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("failed to create history manager: %v", err)
	}

	err = manager.Close()
	if err != nil {
		t.Errorf("failed to close history manager: %v", err)
	}

	// After closing, operations should fail
	err = manager.RecordServerStart("test", "7777")
	if err == nil {
		t.Error("expected error when recording after close, got nil")
	}
}

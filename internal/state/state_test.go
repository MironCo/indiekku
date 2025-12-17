package state

import (
	"testing"
	"time"
)

func TestStateHandler_AddAndGetServer(t *testing.T) {
	handler := NewStateHandler()

	server := &ServerInfo{
		ContainerID:   "abc123",
		ContainerName: "unity-server-7777",
		Port:          "7777",
		PlayerCount:   0,
		StartedAt:     time.Now(),
	}

	handler.AddServer(server)

	retrieved, err := handler.GetServer("unity-server-7777")
	if err != nil {
		t.Fatalf("failed to get server: %v", err)
	}

	if retrieved.ContainerName != server.ContainerName {
		t.Errorf("got container name %q, want %q", retrieved.ContainerName, server.ContainerName)
	}
	if retrieved.Port != server.Port {
		t.Errorf("got port %q, want %q", retrieved.Port, server.Port)
	}
}

func TestStateHandler_RemoveServer(t *testing.T) {
	handler := NewStateHandler()

	server := &ServerInfo{
		ContainerName: "unity-server-7777",
		Port:          "7777",
	}

	handler.AddServer(server)
	handler.RemoveServer("unity-server-7777")

	_, err := handler.GetServer("unity-server-7777")
	if err == nil {
		t.Error("expected error for removed server, got nil")
	}
}

func TestStateHandler_ListServers(t *testing.T) {
	handler := NewStateHandler()

	servers := []*ServerInfo{
		{ContainerName: "server1", Port: "7777"},
		{ContainerName: "server2", Port: "7778"},
		{ContainerName: "server3", Port: "7779"},
	}

	for _, s := range servers {
		handler.AddServer(s)
	}

	list := handler.ListServers()
	if len(list) != 3 {
		t.Errorf("got %d servers, want 3", len(list))
	}
}

func TestStateHandler_UpdatePlayerCount(t *testing.T) {
	handler := NewStateHandler()

	server := &ServerInfo{
		ContainerName: "unity-server-7777",
		Port:          "7777",
		PlayerCount:   0,
	}

	handler.AddServer(server)

	err := handler.UpdatePlayerCount("unity-server-7777", 5)
	if err != nil {
		t.Fatalf("failed to update player count: %v", err)
	}

	retrieved, _ := handler.GetServer("unity-server-7777")
	if retrieved.PlayerCount != 5 {
		t.Errorf("got player count %d, want 5", retrieved.PlayerCount)
	}
}

func TestStateHandler_IsPortInUse(t *testing.T) {
	handler := NewStateHandler()

	server := &ServerInfo{
		ContainerName: "unity-server-7777",
		Port:          "7777",
	}

	handler.AddServer(server)

	if !handler.IsPortInUse("7777") {
		t.Error("port 7777 should be in use")
	}

	if handler.IsPortInUse("7778") {
		t.Error("port 7778 should not be in use")
	}
}

func TestStateHandler_GetNextAvailablePort(t *testing.T) {
	handler := NewStateHandler()

	// Add server on port 7777
	handler.AddServer(&ServerInfo{
		ContainerName: "server1",
		Port:          "7777",
	})

	// Should get 7778 as next available
	nextPort := handler.GetNextAvailablePort(7777)
	if nextPort != "7778" {
		t.Errorf("got next port %q, want 7778", nextPort)
	}

	// Add server on 7778
	handler.AddServer(&ServerInfo{
		ContainerName: "server2",
		Port:          "7778",
	})

	// Should get 7779
	nextPort = handler.GetNextAvailablePort(7777)
	if nextPort != "7779" {
		t.Errorf("got next port %q, want 7779", nextPort)
	}
}

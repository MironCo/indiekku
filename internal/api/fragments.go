package api

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"indiekku/internal/docker"

	"github.com/gin-gonic/gin"
)

// ServeServersFragment returns HTML fragment for the servers table
func (h *ApiHandler) ServeServersFragment(c *gin.Context) {
	servers := h.stateManager.ListServers()

	if len(servers) == 0 {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div class="no-servers">No servers running</div>`))
		return
	}

	var sb strings.Builder
	sb.WriteString(`<table class="servers-table">
    <thead>
        <tr>
            <th>Name</th>
            <th>Port</th>
            <th>Players</th>
            <th>Uptime</th>
            <th>Actions</th>
        </tr>
    </thead>
    <tbody>`)

	for _, server := range servers {
		uptime := formatUptime(server.StartedAt)
		name := html.EscapeString(server.ContainerName)
		port := html.EscapeString(server.Port)

		sb.WriteString(fmt.Sprintf(`
        <tr>
            <td class="server-name">%s</td>
            <td class="server-port">%s</td>
            <td class="server-players">%d</td>
            <td class="server-uptime">%s</td>
            <td>
                <button class="stop-button" onclick="stopServer('%s')">Stop</button>
            </td>
        </tr>`, name, port, server.PlayerCount, uptime, name))
	}

	sb.WriteString(`
    </tbody>
</table>`)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(sb.String()))
}

// ServeServerEventsFragment returns HTML fragment for server events history
func (h *ApiHandler) ServeServerEventsFragment(c *gin.Context) {
	if h.historyManager == nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div class="no-data">History tracking not enabled</div>`))
		return
	}

	events, err := h.historyManager.GetServerEvents("", 100)
	if err != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`<div class="no-data">Error: %s</div>`, html.EscapeString(err.Error()))))
		return
	}

	if len(events) == 0 {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div class="no-data">No server events recorded</div>`))
		return
	}

	var sb strings.Builder
	sb.WriteString(`<table class="history-table">
    <thead>
        <tr>
            <th>Event</th>
            <th>Container</th>
            <th>Port</th>
            <th>Duration</th>
            <th>Timestamp</th>
        </tr>
    </thead>
    <tbody>`)

	for _, event := range events {
		eventType := html.EscapeString(event.EventType)
		containerName := html.EscapeString(event.ContainerName)
		port := html.EscapeString(event.Port)
		duration := "-"
		if event.Duration != nil && *event.Duration > 0 {
			duration = formatDuration(int(*event.Duration))
		}
		timestamp := event.Timestamp.Format("Jan 2, 2006 3:04 PM")

		sb.WriteString(fmt.Sprintf(`
        <tr>
            <td><span class="event-badge %s">%s</span></td>
            <td class="container-name">%s</td>
            <td class="port">%s</td>
            <td class="duration">%s</td>
            <td class="timestamp">%s</td>
        </tr>`, eventType, eventType, containerName, port, duration, timestamp))
	}

	sb.WriteString(`
    </tbody>
</table>`)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(sb.String()))
}

// ServeUploadHistoryFragment returns HTML fragment for upload history
func (h *ApiHandler) ServeUploadHistoryFragment(c *gin.Context) {
	if h.historyManager == nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div class="no-data">History tracking not enabled</div>`))
		return
	}

	uploads, err := h.historyManager.GetUploadHistory(100)
	if err != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`<div class="no-data">Error: %s</div>`, html.EscapeString(err.Error()))))
		return
	}

	if len(uploads) == 0 {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div class="no-data">No upload history recorded</div>`))
		return
	}

	var sb strings.Builder
	sb.WriteString(`<table class="history-table">
    <thead>
        <tr>
            <th>Status</th>
            <th>Filename</th>
            <th>Size</th>
            <th>Timestamp</th>
        </tr>
    </thead>
    <tbody>`)

	for _, upload := range uploads {
		status := "failed"
		if upload.Success {
			status = "success"
		}
		filename := "-"
		if upload.Filename != "" {
			filename = html.EscapeString(upload.Filename)
		}
		filesize := "-"
		if upload.FileSize > 0 {
			filesize = formatBytes(upload.FileSize)
		}
		timestamp := upload.Timestamp.Format("Jan 2, 2006 3:04 PM")

		sb.WriteString(fmt.Sprintf(`
        <tr>
            <td><span class="event-badge %s">%s</span></td>
            <td class="filename">%s</td>
            <td class="filesize">%s</td>
            <td class="timestamp">%s</td>
        </tr>`, status, status, filename, filesize, timestamp))
	}

	sb.WriteString(`
    </tbody>
</table>`)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(sb.String()))
}

// ServeLogsFragment returns HTML fragment for server logs
func (h *ApiHandler) ServeLogsFragment(c *gin.Context) {
	containerName := c.Query("server")
	if containerName == "" {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div class="no-logs">Select a server to view logs</div>`))
		return
	}

	// Import docker package to get logs
	logs, err := getLogs(containerName)
	if err != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`<div class="no-logs">Error: %s</div>`, html.EscapeString(err.Error()))))
		return
	}

	if strings.TrimSpace(logs) == "" {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<div class="no-logs">No logs available</div>`))
		return
	}

	// Return pre-formatted logs
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html.EscapeString(logs)))
}

// ServeServerSelectFragment returns HTML fragment for server dropdown options
func (h *ApiHandler) ServeServerSelectFragment(c *gin.Context) {
	servers := h.stateManager.ListServers()

	if len(servers) == 0 {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<option value="">No servers running</option>`))
		return
	}

	var sb strings.Builder
	sb.WriteString(`<option value="">Select a server...</option>`)

	for _, server := range servers {
		name := html.EscapeString(server.ContainerName)
		port := html.EscapeString(server.Port)
		sb.WriteString(fmt.Sprintf(`<option value="%s">%s (Port: %s)</option>`, name, name, port))
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(sb.String()))
}

// formatUptime formats a duration since start time
func formatUptime(startedAt time.Time) string {
	diff := time.Since(startedAt)

	seconds := int(diff.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours%24)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes%60)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds%60)
	}
	return fmt.Sprintf("%ds", seconds)
}

// formatDuration formats a duration in seconds
func formatDuration(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

// formatBytes formats bytes to human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// getLogs is a helper to get container logs
func getLogs(containerName string) (string, error) {
	logs, err := docker.GetContainerLogsSince(containerName, "5m")
	if err != nil {
		return "", err
	}
	return string(logs), nil
}

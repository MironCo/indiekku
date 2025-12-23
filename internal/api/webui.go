package api

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed webui_index.html
var indexHTML []byte

//go:embed webui_history.html
var historyHTML []byte

// ServeWebUI serves the web UI HTML page
func (h *ApiHandler) ServeWebUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
}

// ServeHistoryUI serves the history page
func (h *ApiHandler) ServeHistoryUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", historyHTML)
}

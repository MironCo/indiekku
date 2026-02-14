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

//go:embed webui_logs.html
var logsHTML []byte

//go:embed webui_styles.css
var stylesCSS []byte

//go:embed webui_deploy.html
var deployHTML []byte

// ServeWebUI serves the web UI HTML page
func (h *ApiHandler) ServeWebUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
}

// ServeHistoryUI serves the history page
func (h *ApiHandler) ServeHistoryUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", historyHTML)
}

// ServeLogsUI serves the logs page
func (h *ApiHandler) ServeLogsUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", logsHTML)
}

// ServeStyles serves the shared CSS stylesheet
func (h *ApiHandler) ServeStyles(c *gin.Context) {
	c.Data(http.StatusOK, "text/css; charset=utf-8", stylesCSS)
}

// ServeDeployUI serves the deploy/new release page
func (h *ApiHandler) ServeDeployUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", deployHTML)
}

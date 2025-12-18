package api

import (
	"github.com/gin-gonic/gin"
)

// ServeWebUI serves the web UI HTML page
func (h *ApiHandler) ServeWebUI(c *gin.Context) {
	c.File("web/index.html")
}

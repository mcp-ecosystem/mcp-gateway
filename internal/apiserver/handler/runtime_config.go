package handler

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// HandleRuntimeConfig serves frontend runtime config as JSON
func HandleRuntimeConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"VITE_API_BASE_URL":         os.Getenv("VITE_API_BASE_URL"),
		"VITE_WS_BASE_URL":          os.Getenv("VITE_WS_BASE_URL"),
		"VITE_MCP_GATEWAY_BASE_URL": os.Getenv("VITE_MCP_GATEWAY_BASE_URL"),
		"VITE_BASE_URL":             os.Getenv("VITE_BASE_URL"),
	})
}

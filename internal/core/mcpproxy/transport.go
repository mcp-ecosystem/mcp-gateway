package mcpproxy

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"
)

// TransportType represents the type of transport
type TransportType string

const (
	// TypeSSE represents SSE-based transport
	TypeSSE TransportType = "sse"
	// TypeStdio represents stdio-based transport
	TypeStdio TransportType = "stdio"
	// TypeStreamable represents streamable HTTP-based transport
	TypeStreamable TransportType = "streamable-http"
)

// Transport defines the interface for MCP transport implementations
type Transport interface {
	// FetchToolList fetches the list of available tools
	FetchToolList(ctx context.Context, conn session.Connection, cfg config.MCPServerConfig) ([]mcp.ToolSchema, error)

	// InvokeTool handles tool invocation
	InvokeTool(c *gin.Context, conn session.Connection, cfg config.MCPServerConfig, params mcp.CallToolParams) (*mcp.CallToolResult, error)
}

// NewTransport creates transport based on the configuration
func NewTransport(cfg config.MCPServerConfig) (Transport, error) {
	switch TransportType(cfg.Type) {
	case TypeSSE:
		return &SSETransport{}, nil
	case TypeStdio:
		return &StdioTransport{}, nil
	case TypeStreamable:
		return &StreamableTransport{}, nil
	default:
		return nil, fmt.Errorf("unknown transport type: %s", cfg.Type)
	}
}

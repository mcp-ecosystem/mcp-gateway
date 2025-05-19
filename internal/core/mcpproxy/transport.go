package mcpproxy

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/internal/template"
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
	FetchToolList(ctx context.Context, conn session.Connection) ([]mcp.ToolSchema, error)

	// InvokeTool handles tool invocation
	InvokeTool(c *gin.Context, conn session.Connection, params mcp.CallToolParams) (*mcp.CallToolResult, error)

	// Start starts the transport
	Start(ctx context.Context, tmplCtx *template.Context) error

	// Stop stops the transport
	Stop(ctx context.Context) error

	// IsStarted returns true if the transport is started
	IsStarted() bool
}

// NewTransport creates transport based on the configuration
func NewTransport(cfg config.MCPServerConfig) (Transport, error) {
	switch TransportType(cfg.Type) {
	case TypeSSE:
		return &SSETransport{cfg: cfg}, nil
	case TypeStdio:
		return &StdioTransport{cfg: cfg}, nil
	case TypeStreamable:
		return &StreamableTransport{cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("unknown transport type: %s", cfg.Type)
	}
}

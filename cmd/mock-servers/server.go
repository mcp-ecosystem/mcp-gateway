package main

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mcp-ecosystem/mcp-gateway/cmd/mock-servers/backend"
)

var logger *zap.Logger

func init() {
	// Initialize logger
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

func StartMockServer(transport string, addr string) {
	switch transport {
	case "http":
		startHTTPServer(addr)
	case "stdio":
		startStdioServer(addr)
	case "sse":
		startSSEServer(addr)
	default:
		logger.Fatal("unsupported transport", zap.String("transport", transport))
	}
}

func startHTTPServer(addr string) {
	backend.NewHTTPServer().Start(addr)
}

func startStdioServer(_ string) {
	mcpServer := backend.NewMCPServer()

	logger.Info("Starting MCP server on stdio")
	if err := server.ServeStdio(mcpServer); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}
}

func startSSEServer(addr string) {
	mcpServer := backend.NewMCPServer()

	sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL(fmt.Sprintf("http://localhost%s", addr)))
	logger.Info("Starting SSE server", zap.String("addr", fmt.Sprintf("http://localhost%s/sse", addr)))
	if err := sseServer.Start(addr); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}
}

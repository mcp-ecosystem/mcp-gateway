package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mcp-ecosystem/mcp-gateway/cmd/mock-server/backend"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/version"
	"github.com/spf13/cobra"
)

var (
	addr    string
	sseAddr string
	logger  *zap.Logger
)

func init() {
	// Initialize logger
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringVarP(&addr, "addr", "a", ":5236", "Address to listen on")
	rootCmd.PersistentFlags().StringVarP(&sseAddr, "sse-addr", "s", ":5237", "Address to listen on for SSE")
}

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of mock-server",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("mock-server version %s\n", version.Get())
		},
	}

	rootCmd = &cobra.Command{
		Use:   "mock-server",
		Short: "Mock Backend Server",
		Long:  `Mock Backend Server provide mock servers for testing`,
		Run: func(cmd *cobra.Command, args []string) {
			StartMockServer(addr)
		},
	}
)

func StartMockServer(addr string) {
	// Create a context that will be canceled on OS signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create error channel to collect errors from all servers
	errChan := make(chan error, 3)

	// Start all servers with context
	go startHTTPServer(ctx, addr, errChan)
	go startStdioServer(ctx, errChan)
	go startSSEServer(ctx, addr, errChan)

	// Wait for either context cancellation or error
	select {
	case <-ctx.Done():
		logger.Info("Received shutdown signal, stopping all servers...")
	case err := <-errChan:
		logger.Error("Server error occurred", zap.Error(err))
		stop() // Cancel context to trigger shutdown
	}

	// Wait for all servers to shutdown
	<-ctx.Done()
	logger.Info("All servers stopped")
}

func startHTTPServer(_ context.Context, addr string, errChan chan<- error) {
	httpServer := backend.NewHTTPServer()
	if err := httpServer.Start(addr); err != nil {
		errChan <- fmt.Errorf("HTTP server error: %w", err)
	}
}

func startStdioServer(_ context.Context, errChan chan<- error) {
	mcpServer := backend.NewMCPServer()

	logger.Info("Starting MCP server on stdio")
	if err := server.ServeStdio(mcpServer); err != nil {
		errChan <- fmt.Errorf("stdio server error: %w", err)
	}
}

func startSSEServer(_ context.Context, addr string, errChan chan<- error) {
	mcpServer := backend.NewMCPServer()

	sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL(fmt.Sprintf("http://localhost%s", sseAddr)))
	logger.Info("Starting SSE server", zap.String("addr", fmt.Sprintf("http://localhost%s/sse", sseAddr)))
	if err := sseServer.Start(sseAddr); err != nil {
		errChan <- fmt.Errorf("SSE server error: %w", err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

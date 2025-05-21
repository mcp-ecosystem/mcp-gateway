package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/core/mcpproxy"
	"github.com/mcp-ecosystem/mcp-gateway/internal/core/state"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type (
	// Server represents the MCP server
	Server struct {
		logger *zap.Logger
		port   int
		router *gin.Engine
		// state contains all the read-only shared state
		state *state.State
		// store is the storage service for MCP configs
		store storage.Store
		// sessions manages all active sessions
		sessions session.Store
		// shutdownCh is used to signal shutdown to all SSE connections
		shutdownCh chan struct{}
		// toolRespHandler is a chain of response handlers
		toolRespHandler ResponseHandler
	}
)

// NewServer creates a new MCP server
func NewServer(logger *zap.Logger, port int, store storage.Store, sessionStore session.Store) (*Server, error) {
	s := &Server{
		logger:          logger,
		port:            port,
		router:          gin.Default(),
		state:           state.NewState(),
		store:           store,
		sessions:        sessionStore,
		shutdownCh:      make(chan struct{}),
		toolRespHandler: CreateResponseHandlerChain(),
	}
	s.router.Use(s.loggerMiddleware())
	s.router.Use(s.recoveryMiddleware())
	return s, nil
}

// RegisterRoutes registers routes with the given router for MCP servers
func (s *Server) RegisterRoutes(ctx context.Context) error {
	s.router.GET("/health_check", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Health check passed.",
		})
	})

	newState, err := s.updateConfigs(ctx)
	if err != nil {
		s.logger.Error("invalid configuration during route registration",
			zap.Error(err))
		return fmt.Errorf("invalid configuration: %w", err)
	}
	// Atomically replace the state
	s.state = newState

	// Register all routes under root path
	s.logger.Debug("registering root handler")
	s.router.NoRoute(s.handleRoot)

	return nil
}

// handleRoot handles all requests and routes them based on the path
func (s *Server) handleRoot(c *gin.Context) {
	path := c.Request.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		s.logger.Debug("invalid path format",
			zap.String("path", path),
			zap.String("remote_addr", c.Request.RemoteAddr))
		s.sendProtocolError(c, nil, "Invalid path", http.StatusBadRequest, mcp.ErrorCodeInvalidRequest)
		return
	}
	endpoint := parts[len(parts)-1]
	prefix := "/" + strings.Join(parts[:len(parts)-1], "/")

	s.logger.Debug("routing request",
		zap.String("path", path),
		zap.String("prefix", prefix),
		zap.String("endpoint", endpoint),
		zap.String("remote_addr", c.Request.RemoteAddr))

	// Dynamically set CORS
	if cors := s.state.GetCORS(prefix); cors != nil {
		s.logger.Debug("applying CORS middleware",
			zap.String("prefix", prefix))
		s.corsMiddleware(cors)(c)
		if c.IsAborted() {
			s.logger.Debug("request aborted by CORS middleware",
				zap.String("prefix", prefix),
				zap.String("remote_addr", c.Request.RemoteAddr))
			return
		}
	}

	protoType := s.state.GetProtoType(prefix)
	if protoType == "" {
		s.logger.Warn("invalid prefix",
			zap.String("prefix", prefix),
			zap.String("remote_addr", c.Request.RemoteAddr))
		s.sendProtocolError(c, nil, "Invalid prefix", http.StatusNotFound, mcp.ErrorCodeInvalidRequest)
		return
	}

	c.Status(http.StatusOK)
	switch endpoint {
	case "sse":
		s.logger.Debug("handling SSE endpoint",
			zap.String("prefix", prefix))
		s.handleSSE(c)
	case "message":
		s.logger.Debug("handling message endpoint",
			zap.String("prefix", prefix))
		s.handleMessage(c)
	case "mcp":
		s.logger.Debug("handling MCP endpoint",
			zap.String("prefix", prefix))
		s.handleMCP(c)
	default:
		s.logger.Warn("invalid endpoint",
			zap.String("endpoint", endpoint),
			zap.String("prefix", prefix),
			zap.String("remote_addr", c.Request.RemoteAddr))
		s.sendProtocolError(c, nil, "Invalid endpoint", http.StatusNotFound, mcp.ErrorCodeInvalidRequest)
	}
}

func (s *Server) Start() {
	go func() {
		if err := s.router.Run(fmt.Sprintf(":%d", s.port)); err != nil {
			s.logger.Error("failed to start server", zap.Error(err))
		}
	}()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(_ context.Context) error {
	s.logger.Info("shutting down server")
	close(s.shutdownCh)

	var wg sync.WaitGroup
	for prefix, transport := range s.state.GetTransports() {
		if transport.IsRunning() {
			wg.Add(1)
			go func(p string, t mcpproxy.Transport) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := t.Stop(ctx); err != nil {
					if err.Error() == "signal: interrupt" {
						s.logger.Info("transport stopped", zap.String("prefix", p))
						return
					}
					s.logger.Error("failed to stop transport",
						zap.String("prefix", p),
						zap.Error(err))
				}
			}(prefix, transport)
		}
	}
	wg.Wait()

	return nil
}

func (s *Server) updateConfigs(ctx context.Context) (*state.State, error) {
	s.logger.Info("Updating MCP configuration")
	//todo we can use hash or version to check if the configuration is changed
	cfgs, err := s.store.List(ctx)
	if err != nil {
		s.logger.Error("Failed to load MCP configurations",
			zap.Error(err))
		return nil, err
	}

	// Validate configurations before merging
	err = config.ValidateMCPConfigs(cfgs)
	if err != nil {
		var validationErr *config.ValidationError
		if errors.As(err, &validationErr) {
			s.logger.Error("Configuration validation failed",
				zap.String("error", validationErr.Error()))
		} else {
			s.logger.Error("failed to validate configurations",
				zap.Error(err))
		}
		return nil, err
	}

	s.logger.Info("initializing server state")
	newState, err := state.BuildStateFromConfig(ctx, cfgs, s.state, s.logger)
	if err != nil {
		s.logger.Error("failed to initialize server state",
			zap.Error(err))
		return nil, err
	}

	s.logger.Info("server configuration loaded",
		zap.Int("server_count", newState.GetServerCount()),
		zap.Int("tool_count", newState.GetToolCount()),
		zap.Int("router_count", newState.GetRouterCount()))

	return newState, nil
}

func (s *Server) ReloadConfigs(ctx context.Context) {
	s.logger.Info("Reloading MCP configuration")

	newState, err := s.updateConfigs(ctx)
	if err != nil {
		s.logger.Error("failed to reload configuration",
			zap.Error(err))
		return
	}
	// Atomically replace the state
	s.state = newState

	s.logger.Info("Configuration reloaded successfully")
}

func (s *Server) UpdateConfig(ctx context.Context, cfg *config.MCPConfig) {
	s.logger.Info("Updating MCP configuration", zap.String("name", cfg.Name))

	// Validate the new configuration
	if err := config.ValidateMCPConfig(cfg); err != nil {
		var validationErr *config.ValidationError
		if errors.As(err, &validationErr) {
			s.logger.Error("Configuration validation failed",
				zap.String("error", validationErr.Error()))
		} else {
			s.logger.Error("failed to validate configuration",
				zap.Error(err))
		}
		return
	}

	// Get current state
	currentState := s.state
	if currentState == nil {
		s.logger.Warn("current state is nil, triggering reload")
		s.ReloadConfigs(ctx)
		return
	}

	// Merge the new configuration with existing configs
	cfgs := config.MergeConfigs(currentState.GetRawConfigs(), cfg)

	// Build new state from updated configs
	updatedState, err := state.BuildStateFromConfig(ctx, cfgs, currentState, s.logger)
	if err != nil {
		s.logger.Error("failed to build state from updated configs",
			zap.Error(err))
		return
	}

	// Log the changes
	s.logger.Info("Configuration updated",
		zap.Int("server_count", updatedState.GetServerCount()),
		zap.Int("tool_count", updatedState.GetToolCount()),
		zap.Int("router_count", updatedState.GetRouterCount()))

	// Atomically replace the state
	s.state = updatedState

	return
}

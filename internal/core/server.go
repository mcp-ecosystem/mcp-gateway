package core

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/core/mcpproxy"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage/helper"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type (
	// Server represents the MCP server
	Server struct {
		logger *zap.Logger
		// state contains all the read-only shared state
		state *serverState
		// sessions manages all active sessions
		sessions session.Store
		// shutdownCh is used to signal shutdown to all SSE connections
		shutdownCh chan struct{}
		// toolRespHandler is a chain of response handlers
		toolRespHandler ResponseHandler
	}

	// serverState contains all the read-only shared state
	serverState struct {
		rawConfigs              []*config.MCPConfig
		tools                   []mcp.ToolSchema
		toolMap                 map[string]*config.ToolConfig
		prefixToTools           map[string][]mcp.ToolSchema
		prefixToServerConfig    map[string]*config.ServerConfig
		prefixToRouterConfig    map[string]*config.RouterConfig
		prefixToMCPServerConfig map[string]config.MCPServerConfig
		prefixToProtoType       map[string]cnst.ProtoType
		prefixToTransport       map[string]mcpproxy.Transport
	}
)

// NewServer creates a new MCP server
func NewServer(logger *zap.Logger, cfg *config.MCPGatewayConfig) (*Server, error) {
	// Initialize session store
	sessionStore, err := session.NewStore(logger, &cfg.Session)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session store: %w", err)
	}

	return &Server{
		logger: logger,
		state: &serverState{
			rawConfigs:              make([]*config.MCPConfig, 0),
			tools:                   make([]mcp.ToolSchema, 0),
			toolMap:                 make(map[string]*config.ToolConfig),
			prefixToTools:           make(map[string][]mcp.ToolSchema),
			prefixToServerConfig:    make(map[string]*config.ServerConfig),
			prefixToRouterConfig:    make(map[string]*config.RouterConfig),
			prefixToMCPServerConfig: make(map[string]config.MCPServerConfig),
			prefixToProtoType:       make(map[string]cnst.ProtoType),
			prefixToTransport:       make(map[string]mcpproxy.Transport),
		},
		sessions:        sessionStore,
		shutdownCh:      make(chan struct{}),
		toolRespHandler: CreateResponseHandlerChain(),
	}, nil
}

// RegisterRoutes registers routes with the given router for MCP servers
func (s *Server) RegisterRoutes(router *gin.Engine, cfgs []*config.MCPConfig) error {
	// Validate configuration before registering routes
	if err := config.ValidateMCPConfigs(cfgs); err != nil {
		s.logger.Error("invalid configuration during route registration",
			zap.Error(err))
		return fmt.Errorf("invalid configuration: %w", err)
	}

	s.logger.Info("registering middleware")
	router.Use(s.loggerMiddleware())
	router.Use(s.recoveryMiddleware())

	// Create new state and load configuration
	s.logger.Debug("initializing server state")
	newState, err := initState(cfgs, s.state)
	if err != nil {
		s.logger.Error("failed to initialize server state",
			zap.Error(err))
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// 记录配置信息
	s.logger.Info("server configuration loaded",
		zap.Int("server_count", len(newState.prefixToServerConfig)),
		zap.Int("tool_count", len(newState.toolMap)),
		zap.Int("router_count", len(newState.prefixToRouterConfig)))

	// Atomically replace the state
	s.state = newState

	// Register all routes under root path
	s.logger.Debug("registering root handler")
	router.NoRoute(s.handleRoot)

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
	if routerCfg, ok := s.state.prefixToRouterConfig[prefix]; ok && routerCfg.CORS != nil {
		s.logger.Debug("applying CORS middleware",
			zap.String("prefix", prefix))
		s.corsMiddleware(routerCfg.CORS)(c)
		if c.IsAborted() {
			s.logger.Debug("request aborted by CORS middleware",
				zap.String("prefix", prefix),
				zap.String("remote_addr", c.Request.RemoteAddr))
			return
		}
	}

	state := s.state
	if _, ok := state.prefixToProtoType[prefix]; !ok {
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

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(_ context.Context) error {
	s.logger.Info("shutting down server")
	close(s.shutdownCh)
	return nil
}

// initState creates a new serverState from the given configuration
func initState(cfgs []*config.MCPConfig, oldState *serverState) (*serverState, error) {
	// Create new state
	newState := &serverState{
		rawConfigs:              cfgs,
		tools:                   make([]mcp.ToolSchema, 0),
		toolMap:                 make(map[string]*config.ToolConfig),
		prefixToTools:           make(map[string][]mcp.ToolSchema),
		prefixToServerConfig:    make(map[string]*config.ServerConfig),
		prefixToRouterConfig:    make(map[string]*config.RouterConfig),
		prefixToMCPServerConfig: make(map[string]config.MCPServerConfig),
		prefixToProtoType:       make(map[string]cnst.ProtoType),
		prefixToTransport:       make(map[string]mcpproxy.Transport),
	}

	for idx := range cfgs {
		cfg := cfgs[idx]

		// Initialize tool map and list for MCP servers
		for i := range cfg.Tools {
			tool := &cfg.Tools[i]
			newState.toolMap[tool.Name] = tool
			newState.tools = append(newState.tools, tool.ToToolSchema())
		}

		// Build prefix to tools mapping for MCP servers
		prefixMap := make(map[string]string)
		for i, routerCfg := range cfg.Routers {
			prefixMap[routerCfg.Server] = routerCfg.Prefix
			newState.prefixToRouterConfig[routerCfg.Prefix] = &cfg.Routers[i]
		}

		// Process regular HTTP servers
		for _, serverCfg := range cfg.Servers {
			prefix, exists := prefixMap[serverCfg.Name]
			if !exists {
				return nil, fmt.Errorf("no router prefix found for MCP server: %s", serverCfg.Name)
			}

			// Filter tools based on MCP server's allowed tools
			var allowedTools []mcp.ToolSchema
			for _, toolName := range serverCfg.AllowedTools {
				if tool, ok := newState.toolMap[toolName]; ok {
					allowedTools = append(allowedTools, tool.ToToolSchema())
				}
			}
			newState.prefixToTools[prefix] = allowedTools
			newState.prefixToServerConfig[prefix] = &serverCfg
			newState.prefixToProtoType[prefix] = cnst.BackendProtoHttp
		}

		// Process MCP servers
		for _, mcpServer := range cfg.McpServers {
			prefix, exists := prefixMap[mcpServer.Name]
			if !exists {
				continue // Skip MCP servers without router prefix
			}

			// Map prefix to MCP server config
			newState.prefixToMCPServerConfig[prefix] = mcpServer

			// Check if we already have transport with the same configuration
			var transport mcpproxy.Transport
			if oldState != nil {
				if oldTransport, exists := oldState.prefixToTransport[prefix]; exists {
					// Compare configurations to see if we need to create a new transport
					oldConfig := oldState.prefixToMCPServerConfig[prefix]
					if oldConfig.Type == mcpServer.Type &&
						oldConfig.Command == mcpServer.Command &&
						oldConfig.URL == mcpServer.URL &&
						len(oldConfig.Args) == len(mcpServer.Args) {
						// Compare args
						argsMatch := true
						for i, arg := range oldConfig.Args {
							if arg != mcpServer.Args[i] {
								argsMatch = false
								break
							}
						}
						if argsMatch {
							// Reuse existing transport
							transport = oldTransport
						}
					}
				}
			}

			// Create new transport if needed
			if transport == nil {
				var err error
				transport, err = mcpproxy.NewTransport(mcpServer)
				if err != nil {
					return nil, fmt.Errorf("failed to create transport for server %s: %w", mcpServer.Name, err)
				}
			}
			newState.prefixToTransport[prefix] = transport

			// Map protocol type based on server type
			switch mcpServer.Type {
			case "stdio":
				newState.prefixToProtoType[prefix] = cnst.BackendProtoStdio
			case "sse":
				newState.prefixToProtoType[prefix] = cnst.BackendProtoSSE
			case "streamable-http":
				newState.prefixToProtoType[prefix] = cnst.BackendProtoStreamable
			}
		}
	}

	return newState, nil
}

// UpdateConfig updates the server configuration
func (s *Server) UpdateConfig(cfgs []*config.MCPConfig) error {
	// Validate configuration before updating
	s.logger.Debug("validating updated configuration")
	if err := config.ValidateMCPConfigs(cfgs); err != nil {
		s.logger.Error("invalid configuration during update",
			zap.Error(err))
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create new state and load configuration
	s.logger.Info("updating server configuration")
	newState, err := initState(cfgs, s.state)
	if err != nil {
		s.logger.Error("failed to initialize state during update",
			zap.Error(err))
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// 记录配置更新信息
	s.logger.Info("server configuration updated",
		zap.Int("server_count", len(newState.prefixToServerConfig)),
		zap.Int("tool_count", len(newState.toolMap)),
		zap.Int("router_count", len(newState.prefixToRouterConfig)))

	// Atomically replace the state
	s.state = newState

	return nil
}

// MergeConfig updates the server configuration incrementally
func (s *Server) MergeConfig(cfg *config.MCPConfig) error {
	s.logger.Info("merging configuration")

	newConfig, err := helper.MergeConfigs(s.state.rawConfigs, cfg)
	if err != nil {
		s.logger.Error("failed to merge configuration",
			zap.Error(err))
		return fmt.Errorf("failed to merge configuration: %w", err)
	}

	// Validate configuration after merge
	s.logger.Debug("validating merged configuration")
	if err := config.ValidateMCPConfig(cfg); err != nil {
		s.logger.Error("invalid configuration after merge",
			zap.Error(err))
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create new state and load configuration
	s.logger.Debug("initializing state with merged configuration")
	newState, err := initState(newConfig, s.state)
	if err != nil {
		s.logger.Error("failed to initialize state with merged configuration",
			zap.Error(err))
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Record configuration merge information
	s.logger.Info("configuration merged successfully",
		zap.Int("server_count", len(newState.prefixToServerConfig)),
		zap.Int("tool_count", len(newState.toolMap)),
		zap.Int("router_count", len(newState.prefixToRouterConfig)))

	// Atomically replace the state
	s.state = newState

	return nil
}

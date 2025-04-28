package core

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/template"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"
	"go.uber.org/zap"
)

type (
	// Server represents the MCP server
	Server struct {
		logger   *zap.Logger
		renderer *template.Renderer
		sessions sync.Map
		tools    []mcp.ToolSchema
		toolMap  map[string]*config.ToolConfig
		// prefixToTools maps prefix to allowed tools for each MCP server
		prefixToTools map[string][]mcp.ToolSchema
		// prefixToServerConfig maps prefix to server config for each MCP server
		prefixToServerConfig map[string]*config.ServerConfig
		// sessionToPrefix maps session ID to MCP server prefix
		sessionToPrefix sync.Map

		// sessionStore manages all active sessions
		sessionStore   session.Store
		memorySessions map[string]*sessionDataInMemory
		sLock          sync.RWMutex
	}

	sessionDataInMemory struct {
		flusher http.Flusher
		conn    session.Connection
		meta    *session.Meta
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
		logger:               logger,
		renderer:             template.NewRenderer(),
		tools:                make([]mcp.ToolSchema, 0),
		toolMap:              make(map[string]*config.ToolConfig),
		prefixToTools:        make(map[string][]mcp.ToolSchema),
		prefixToServerConfig: make(map[string]*config.ServerConfig),
		sessionStore:         sessionStore,
		memorySessions:       make(map[string]*sessionDataInMemory),
	}, nil
}

// RegisterRoutes registers routes with the given router for MCP servers
func (s *Server) RegisterRoutes(router *gin.Engine, cfg *config.MCPConfig) error {
	router.Use(s.loggerMiddleware())
	router.Use(s.recoveryMiddleware())

	// Initialize tool map and list for MCP servers
	s.LoadConfig(cfg)

	// Build prefix to tools mapping for MCP servers
	prefixMap := make(map[string]string)
	routerConfigs := make(map[string]*config.RouterConfig)
	for _, routerCfg := range cfg.Routers {
		prefixMap[routerCfg.Server] = routerCfg.Prefix
		routerConfigs[routerCfg.Prefix] = &routerCfg
	}

	for _, serverCfg := range cfg.Servers {
		prefix, exists := prefixMap[serverCfg.Name]
		if !exists {
			return fmt.Errorf("no router prefix found for MCP server: %s", serverCfg.Name)
		}

		// Filter tools based on MCP server's allowed tools
		var allowedTools []mcp.ToolSchema
		for _, toolName := range serverCfg.AllowedTools {
			if tool, ok := s.toolMap[toolName]; ok {
				allowedTools = append(allowedTools, tool.ToToolSchema())
			}
		}
		s.prefixToTools[prefix] = allowedTools
		s.prefixToServerConfig[prefix] = &serverCfg

		group := router.Group(prefix)

		// Add CORS middleware if configured in router
		if routerCfg, ok := routerConfigs[prefix]; ok && routerCfg.CORS != nil {
			group.Use(s.corsMiddleware(routerCfg.CORS))
		}

		// Add both old SSE endpoints and new MCP endpoint
		group.GET("/sse", s.handleSSE)
		group.POST("/message", s.handleMessage)
		group.Any("/mcp", s.handleMCP)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(_ context.Context) error {
	return nil
}

// LoadConfig loads the MCP server configuration
func (s *Server) LoadConfig(cfg *config.MCPConfig) {
	// Initialize tool map and list for MCP servers
	for i := range cfg.Tools {
		tool := &cfg.Tools[i]
		s.toolMap[tool.Name] = tool
		s.tools = append(s.tools, tool.ToToolSchema())
	}
}

// UpdateConfig updates the server configuration
func (s *Server) UpdateConfig(cfg *config.MCPConfig) error {
	// Clear existing tools
	s.tools = make([]mcp.ToolSchema, 0)
	s.toolMap = make(map[string]*config.ToolConfig)
	s.prefixToTools = make(map[string][]mcp.ToolSchema)

	// Initialize tool map and list for MCP servers
	s.LoadConfig(cfg)

	// Build prefix to tools mapping for MCP servers
	prefixMap := make(map[string]string)
	for _, routerCfg := range cfg.Routers {
		prefixMap[routerCfg.Server] = routerCfg.Prefix
	}

	for _, serverCfg := range cfg.Servers {
		prefix, exists := prefixMap[serverCfg.Name]
		if !exists {
			return fmt.Errorf("no router prefix found for MCP server: %s", serverCfg.Name)
		}

		// Filter tools based on MCP server's allowed tools
		var allowedTools []mcp.ToolSchema
		for _, toolName := range serverCfg.AllowedTools {
			if tool, ok := s.toolMap[toolName]; ok {
				allowedTools = append(allowedTools, tool.ToToolSchema())
			}
		}
		s.prefixToTools[prefix] = allowedTools
	}

	return nil
}

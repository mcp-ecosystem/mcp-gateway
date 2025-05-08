package core

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
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
	}

	// serverState contains all the read-only shared state
	serverState struct {
		tools                []mcp.Tool
		toolMap              map[string]*config.ToolConfig
		stdioMap             map[string]*config.StdioConfig
		sseMap               map[string]*config.SSEConfig
		prefixToTools        map[string][]mcp.Tool
		prefixToServerConfig map[string]*config.ServerConfig
		prefixToRouterConfig map[string]*config.RouterConfig
		prefixToProtoType    map[string]cnst.ProtoType
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
			tools:                make([]mcp.Tool, 0),
			toolMap:              make(map[string]*config.ToolConfig),
			prefixToTools:        make(map[string][]mcp.Tool),
			prefixToServerConfig: make(map[string]*config.ServerConfig),
			prefixToRouterConfig: make(map[string]*config.RouterConfig),
		},
		sessions:   sessionStore,
		shutdownCh: make(chan struct{}),
	}, nil
}

// RegisterRoutes registers routes with the given router for MCP servers
func (s *Server) RegisterRoutes(router *gin.Engine, cfgs []*config.MCPConfig) error {
	// Validate configuration before registering routes
	if err := config.ValidateMCPConfigs(cfgs); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	router.Use(s.loggerMiddleware())
	router.Use(s.recoveryMiddleware())

	// Create new state and load configuration
	newState, err := loadConfig(cfgs)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Atomically replace the state
	s.state = newState

	// Register all routes under root path
	router.Any("/*path", s.handleRoot)

	return nil
}

// handleRoot handles all requests and routes them based on the path
func (s *Server) handleRoot(c *gin.Context) {
	path := c.Request.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		s.sendProtocolError(c, nil, "Invalid path", http.StatusBadRequest, mcp.ErrorCodeInvalidRequest)
		return
	}
	endpoint := parts[len(parts)-1]
	prefix := "/" + strings.Join(parts[:len(parts)-1], "/")

	// 动态设置CORS
	if routerCfg, ok := s.state.prefixToRouterConfig[prefix]; ok && routerCfg.CORS != nil {
		s.corsMiddleware(routerCfg.CORS)(c)
		if c.IsAborted() {
			return
		}
	}

	state := s.state
	if _, ok := state.prefixToProtoType[prefix]; !ok {
		s.sendProtocolError(c, nil, "Invalid prefix", http.StatusNotFound, mcp.ErrorCodeInvalidRequest)
		return
	}

	switch endpoint {
	case "sse":
		s.handleSSE(c)
	case "message":
		s.handleMessage(c)
	case "mcp":
		s.handleMCP(c)
	default:
		s.sendProtocolError(c, nil, "Invalid endpoint", http.StatusNotFound, mcp.ErrorCodeInvalidRequest)
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(_ context.Context) error {
	close(s.shutdownCh)
	return nil
}

// loadConfig creates a new serverState from the given configuration
func loadConfig(cfgs []*config.MCPConfig) (*serverState, error) {
	// Create new state
	newState := &serverState{
		tools:                make([]mcp.Tool, 0),
		toolMap:              make(map[string]*config.ToolConfig),
		stdioMap:             make(map[string]*config.StdioConfig),
		sseMap:               make(map[string]*config.SSEConfig),
		prefixToTools:        make(map[string][]mcp.Tool),
		prefixToServerConfig: make(map[string]*config.ServerConfig),
		prefixToRouterConfig: make(map[string]*config.RouterConfig),
		prefixToProtoType:    make(map[string]cnst.ProtoType),
	}

	for idx := range cfgs {
		cfg := cfgs[idx]

		// Build prefix to tools mapping for MCP servers
		prefixMap := make(map[string]string)
		for i, routerCfg := range cfg.Routers {
			prefixMap[routerCfg.Server] = routerCfg.Prefix
			newState.prefixToRouterConfig[routerCfg.Prefix] = &cfg.Routers[i]
		}

		for _, serverCfg := range cfg.Servers {
			prefix, exists := prefixMap[serverCfg.Name]
			if !exists {
				return nil, fmt.Errorf("no router prefix found for MCP server: %s", serverCfg.Name)
			}

			// Filter tools based on MCP server's allowed tools
			var allowedTools []mcp.Tool
			for _, toolName := range serverCfg.AllowedTools {
				if tool, ok := newState.toolMap[toolName]; ok {
					allowedTools = append(allowedTools, tool.ToToolSchema())
				}
			}
			newState.prefixToTools[prefix] = allowedTools
			newState.prefixToServerConfig[prefix] = &serverCfg
			newState.prefixToProtoType[prefix] = cfg.ProtoType
		}

		// Initialize tool map and list for MCP servers
		for i := range cfg.Tools {
			tool := &cfg.Tools[i]
			newState.toolMap[tool.Name] = tool
			newState.tools = append(newState.tools, tool.ToToolSchema())
		}

		// Initialize stdio map and list for MCP servers
		for j := range cfg.StdioConfigs {
			stdioConfig := &cfg.StdioConfigs[j]
			newState.stdioMap[stdioConfig.Name] = stdioConfig
		}

		// Initialize sse map and list for MCP servers
		for k := range cfg.SSEConfigs {
			sseConfig := &cfg.SSEConfigs[k]
			newState.sseMap[sseConfig.Name] = sseConfig
		}
	}

	return newState, nil
}

// UpdateConfig updates the server configuration
func (s *Server) UpdateConfig(cfgs []*config.MCPConfig) error {
	// Validate configuration before updating
	if err := config.ValidateMCPConfigs(cfgs); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create new state and load configuration
	newState, err := loadConfig(cfgs)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Atomically replace the state
	s.state = newState

	return nil
}

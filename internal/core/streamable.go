package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// handleMCP handles MCP connections
func (s *Server) handleMCP(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodOptions:
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Status(http.StatusOK)
		return

	case http.MethodGet:
		prefix := strings.TrimSuffix(c.Request.URL.Path, "/mcp")
		if prefix == "" {
			prefix = "/"
		}

		sessionID := c.GetHeader("Mcp-Session-Id")
		if sessionID == "" {
			sessionID = uuid.New().String()
			c.Header("Mcp-Session-Id", sessionID)
		}

		// TODO: complete
	case http.MethodPost:
		var req mcp.JSONRPCRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			s.sendProtocolError(c, nil, mcp.ErrorCodeParseError, "Invalid JSON-RPC request", http.StatusBadRequest)
			return
		}

		sessionID := c.GetHeader("Mcp-Session-Id")
		if sessionID == "" {
			sessionID = uuid.New().String()
			c.Header("Mcp-Session-Id", sessionID)
		}

		var (
			conn session.Connection
			err  error
		)
		if req.Method == mcp.Initialize {
			prefix := strings.TrimSuffix(c.Request.URL.Path, "/mcp")
			if prefix == "" {
				prefix = "/"
			}

			meta := &session.Meta{
				ID:        sessionID,
				CreatedAt: time.Now(),
				Prefix:    prefix,
				Type:      "streamable",
			}
			conn, err = s.sessions.Register(c.Request.Context(), meta)
			if err != nil {
				s.sendProtocolError(c, req.Id, mcp.ErrorCodeInternalError, "Failed to create session", http.StatusInternalServerError)
				return
			}
		} else {
			conn, err = s.sessions.Get(c.Request.Context(), sessionID)
			if err != nil {
				s.sendProtocolError(c, req.Id, mcp.ErrorCodeInvalidRequest, "Invalid or expired session", http.StatusBadRequest)
				return
			}
		}

		if err := s.handleMCPRequest(c, req, conn); err != nil {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeInternalError, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	case http.MethodDelete:
		sessionID := c.GetHeader("Mcp-Session-Id")
		if sessionID == "" {
			s.sendProtocolError(c, nil, mcp.ErrorCodeInvalidRequest, "Missing session ID", http.StatusBadRequest)
			return
		}

		err := s.sessions.Unregister(c.Request.Context(), sessionID)
		if err != nil {
			s.sendProtocolError(c, nil, mcp.ErrorCodeInvalidRequest, "Invalid or expired session", http.StatusBadRequest)
			return
		}
		c.String(http.StatusOK, "Session terminated")

	default:
		c.Header("Allow", "GET, POST, DELETE")
		s.sendProtocolError(c, nil, mcp.ErrorCodeConnectionClosed, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) handleMCPRequest(c *gin.Context, req mcp.JSONRPCRequest, conn session.Connection) error {
	// Process the request based on its method
	switch req.Method {
	case mcp.Initialize:
		// Handle initialization request
		var params mcp.InitializeRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeInvalidParams, fmt.Sprintf("invalid initialize parameters: %v", err), http.StatusBadRequest)
			return nil
		}

		s.sendSuccessResponse(c, conn, req, mcp.InitializedResult{
			ProtocolVersion: mcp.LatestProtocolVersion,
			Capabilities: mcp.ServerCapabilitiesSchema{
				Logging: mcp.LoggingCapabilitySchema{},
				Tools: mcp.ToolsCapabilitySchema{
					ListChanged: true,
				},
			},
			ServerInfo: mcp.ImplementationSchema{
				Name:    "mcp-gateway",
				Version: "1.0.0",
			},
		}, false)
		return nil

	case mcp.NotificationInitialized:
		c.Header("Content-Type", "application/json")
		c.String(http.StatusAccepted, mcp.Accepted)
		return nil

	case mcp.ToolsList:
		// Get tools for this prefix
		tools, ok := s.prefixToTools[conn.Meta().Prefix]
		if !ok {
			tools = []mcp.ToolSchema{}
		}

		s.sendSuccessResponse(c, conn, req, mcp.ListToolsResult{
			Tools: tools,
		}, false)
		return nil

	case mcp.ToolsCall:
		// Handle tool call request
		var params mcp.CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeInvalidParams, fmt.Sprintf("invalid tool call parameters: %v", err), http.StatusBadRequest)
			return nil
		}

		// Find the tool in the precomputed map
		tool, exists := s.toolMap[params.Name]
		if !exists {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeMethodNotFound, "Tool not found", http.StatusNotFound)
			return nil
		}

		// Convert arguments to map[string]any
		var args map[string]any
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeInvalidParams, "Invalid tool arguments", http.StatusBadRequest)
			return nil
		}

		serverCfg, ok := s.prefixToServerConfig[conn.Meta().Prefix]
		if !ok {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeInternalError, "Server config not found", http.StatusInternalServerError)
			return nil
		}

		// Execute the tool
		result, err := s.executeTool(tool, args, c.Request, serverCfg.Config)
		if err != nil {
			s.logger.Error("failed to execute tool", zap.Error(err))
			// For tool execution errors, return them in the result with isError=true
			s.sendToolExecutionError(c, conn, req, err, false)
			return nil
		}

		s.sendSuccessResponse(c, conn, req, mcp.CallToolResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: result,
				},
			},
			IsError: false,
		}, false)
		return nil

	default:
		s.sendProtocolError(c, req.Id, mcp.ErrorCodeMethodNotFound, "Method not found", http.StatusNotFound)
		return nil
	}
}

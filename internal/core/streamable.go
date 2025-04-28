package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"

	"github.com/gin-gonic/gin"
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
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		sessionID := c.GetHeader("Mcp-Session-Id")
		if sessionID == "" {
			sessionID = uuid.New().String()
			c.Header("Mcp-Session-Id", sessionID)
		}

		var sess *sessionDataInMemory
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
			conn, err := s.sessionStore.Register(c.Request.Context(), meta)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to create session")
				return
			}

			sess = &sessionDataInMemory{
				conn: conn,
				meta: meta,
			}
			s.sessionToPrefix.Store(sessionID, prefix)
		} else {
			conn, err := s.sessionStore.Get(c.Request.Context(), sessionID)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to get session")
				return
			}
			sess = &sessionDataInMemory{
				conn: conn,
				meta: conn.Meta(),
			}
		}

		if err := s.handleMCPRequest(c, req, sess); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		return
	case http.MethodDelete:
		sessionID := c.GetHeader("Mcp-Session-Id")
		if sessionID == "" {
			c.String(http.StatusBadRequest, "Missing session ID")
			return
		}

		err := s.sessionStore.Unregister(c.Request.Context(), sessionID)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid or expired session")
			return
		}
		s.sessionToPrefix.Delete(sessionID)

		c.String(http.StatusOK, "Session terminated")

	default:
		c.String(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleMCPRequest(c *gin.Context, req mcp.JSONRPCRequest, sess *sessionDataInMemory) error {
	// Process the request based on its method
	switch req.Method {
	case mcp.Initialize:
		// Handle initialization request
		var params mcp.InitializeRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendErrorResponse(c, sess, req, fmt.Sprintf("invalid initialize parameters: %v", err))
			return nil
		}

		response := mcp.InitializeResult{
			JSONRPCBaseResult: mcp.JSONRPCBaseResult{
				JSONRPC: mcp.JSPNRPCVersion,
				ID:      req.Id,
			},
			Result: mcp.InitializedResult{
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
			},
		}

		eventData, err := json.Marshal(response)
		if err != nil {
			s.sendErrorResponse(c, sess, req, fmt.Sprintf("failed to marshal response: %v", err))
			return nil
		}

		// Send response directly
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		// TODO: maybe we need to send this to the session store too.
		c.String(http.StatusOK, fmt.Sprintf("event: message\ndata: %s\n\n", eventData))
		return nil

	case mcp.NotificationInitialized:
		// Set response headers
		c.Header("X-Powered-By", "Express")
		c.Header("Connection", "keep-alive")
		c.Header("Keep-Alive", "timeout=5")
		c.Header("Transfer-Encoding", "chunked")

		// Return accepted response
		c.String(http.StatusAccepted, mcp.Accepted)
		return nil

	case mcp.ToolsList:
		// Get the prefix for this session
		prefixI, ok := s.sessionToPrefix.Load(sess.meta.ID)
		if !ok {
			s.sendErrorResponse(c, sess, req, "Session not found")
			return nil
		}
		prefix := prefixI.(string)

		// Get tools for this prefix
		tools, ok := s.prefixToTools[prefix]
		if !ok {
			tools = []mcp.ToolSchema{}
		}

		response := mcp.JSONRPCResponse{
			JSONRPCBaseResult: mcp.JSONRPCBaseResult{
				JSONRPC: mcp.JSPNRPCVersion,
				ID:      req.Id,
			},
			Result: mcp.ListToolsResult{
				Tools: tools,
			},
		}

		eventData, err := json.Marshal(response)
		if err != nil {
			s.sendErrorResponse(c, sess, req, fmt.Sprintf("failed to marshal response: %v", err))
			return nil
		}

		// Send response directly
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.String(http.StatusOK, fmt.Sprintf("event: message\ndata: %s\n\n", eventData))
		return nil

	case mcp.ToolsCall:
		// Handle tool call request
		var params mcp.CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendErrorResponse(c, sess, req, fmt.Sprintf("invalid tool call parameters: %v", err))
			return nil
		}

		// Find the tool in the precomputed map
		tool, exists := s.toolMap[params.Name]
		if !exists {
			s.sendErrorResponse(c, sess, req, "Tool not found")
			return nil
		}

		// Convert arguments to map[string]any
		var args map[string]any
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendErrorResponse(c, sess, req, "Invalid tool arguments")
			return nil
		}

		prefixI, ok := s.sessionToPrefix.Load(sess.meta.ID)
		if !ok {
			s.sendErrorResponse(c, sess, req, "Session not found")
			return nil
		}
		prefix := prefixI.(string)
		serverCfg, ok := s.prefixToServerConfig[prefix]

		// Execute the tool
		result, err := s.executeTool(tool, args, c.Request, serverCfg.Config)
		if err != nil {
			s.sendErrorResponse(c, sess, req, fmt.Sprintf("Error: %s", err.Error()))
			return nil
		}

		response := mcp.JSONRPCResponse{
			JSONRPCBaseResult: mcp.JSONRPCBaseResult{
				JSONRPC: mcp.JSPNRPCVersion,
				ID:      req.Id,
			},
			Result: mcp.CallToolResult{
				Content: []mcp.Content{
					{
						Type: "text",
						Text: result,
					},
				},
			},
		}

		eventData, err := json.Marshal(response)
		if err != nil {
			s.sendErrorResponse(c, sess, req, fmt.Sprintf("failed to marshal response: %v", err))
			return nil
		}

		// Send response directly
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.String(http.StatusOK, fmt.Sprintf("event: message\ndata: %s\n\n", eventData))
		return nil

	default:
		s.sendErrorResponse(c, sess, req, fmt.Sprintf("unknown method: %s", req.Method))
		return nil
	}
}

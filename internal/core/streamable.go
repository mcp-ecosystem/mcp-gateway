package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/version"

	"github.com/google/uuid"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

// handleMCP handles MCP connections
func (s *Server) handleMCP(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodOptions:
		c.Status(http.StatusOK)
		return

	case http.MethodGet:
		s.handleGet(c)
	case http.MethodPost:
		s.handlePost(c)
		return
	case http.MethodDelete:
		s.handleDelete(c)
		return

	default:
		c.Header("Allow", "GET, POST, DELETE")
		s.sendProtocolError(c, nil, "Method not allowed", http.StatusMethodNotAllowed, mcp.ErrorCodeConnectionClosed)
		return
	}
}

// handleGet handles GET requests for SSE stream
func (s *Server) handleGet(c *gin.Context) {
	// Check Accept header for text/event-stream
	acceptHeader := c.GetHeader("Accept")
	if !strings.Contains(acceptHeader, "text/event-stream") {
		s.sendProtocolError(c, nil, "Not Acceptable: Client must accept text/event-stream", http.StatusNotAcceptable, mcp.ErrorCodeInvalidRequest)
		return
	}

	conn := s.getSession(c)
	if conn == nil {
		return
	}

	// TODO: replay events according to the last-event-id in request headers.

	// TODO: only support one sse stream per session, we can detect it when sub the redis topic

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set(mcp.HeaderMcpSessionID, conn.Meta().ID)
	c.Writer.Flush()

	for {
		select {
		case event := <-conn.EventQueue():
			switch event.Event {
			case "message":
				_, err := fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", event.Data)
				if err != nil {
					s.logger.Error("failed to send SSE message", zap.Error(err))
				}
			}
			_, _ = fmt.Fprint(c.Writer, event)
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		case <-s.shutdownCh:
			return
		}
	}
}

// handlePost handles POST requests containing JSON-RPC messages
func (s *Server) handlePost(c *gin.Context) {
	// Validate Accept header
	accept := c.GetHeader("Accept")
	if !(strings.Contains(accept, "application/json") || strings.Contains(accept, "*/*")) ||
		!(strings.Contains(accept, "text/event-stream") || strings.Contains(accept, "*/*")) {
		s.sendProtocolError(c, nil,
			"Not Acceptable: Client must accept both application/json and text/event-stream",
			http.StatusNotAcceptable, mcp.ErrorCodeConnectionClosed)
		return
	}

	// Validate Content-Type header
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		s.sendProtocolError(c, nil, "Unsupported Media Type: Content-Type must be application/json",
			http.StatusUnsupportedMediaType, mcp.ErrorCodeConnectionClosed)
		return
	}

	// TODO: support batch messages
	var req mcp.JSONRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendProtocolError(c, nil, "Invalid JSON-RPC request",
			http.StatusBadRequest, mcp.ErrorCodeParseError)
		return
	}

	sessionID := c.GetHeader(mcp.HeaderMcpSessionID)

	var (
		conn session.Connection
		err  error
	)
	if req.Method == mcp.Initialize {
		if sessionID != "" {
			// confirm if it's registered
			conn, err = s.sessions.Get(c.Request.Context(), sessionID)
			if err != nil {
				s.sendProtocolError(c, req.Id, "Failed to get session", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}
			if conn != nil {
				s.sendProtocolError(c, req.Id, "Invalid Request: Server already initialized", http.StatusBadRequest, mcp.ErrorCodeInvalidRequest)
				return
			}
		} else {
			sessionID = uuid.New().String()
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
				s.sendProtocolError(c, req.Id, "Failed to create session", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}
		}
		c.Header(mcp.HeaderMcpSessionID, sessionID)
	} else {
		conn, err = s.sessions.Get(c.Request.Context(), sessionID)
		if err != nil {
			s.sendProtocolError(c, req.Id, "Invalid Request: Session not found", http.StatusBadRequest, mcp.ErrorCodeInvalidRequest)
			return
		}
	}

	s.handleMCPRequest(c, req, conn)
}

// handleDelete handles DELETE requests to terminate sessions
func (s *Server) handleDelete(c *gin.Context) {
	conn := s.getSession(c)
	if conn == nil {
		return
	}

	err := s.sessions.Unregister(c.Request.Context(), conn.Meta().ID)
	if err != nil {
		s.sendProtocolError(c, conn.Meta().ID, "Failed to terminate session", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
		return
	}
	c.Status(http.StatusOK)
}

func (s *Server) handleMCPRequest(c *gin.Context, req mcp.JSONRPCRequest, conn session.Connection) {
	// Process the request based on its method
	switch req.Method {
	case mcp.Initialize:
		// Handle initialization request
		var params mcp.InitializeRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendProtocolError(c, req.Id, fmt.Sprintf("invalid initialize parameters: %v", err), http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
			return
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
				Name:    cnst.AppName,
				Version: version.Get(),
			},
		}, false)
		return

	case mcp.NotificationInitialized:
		c.Status(http.StatusAccepted)
		return

	case mcp.ToolsList:
		protoType, ok := s.state.prefixToProtoType[conn.Meta().Prefix]
		if !ok {
			s.sendProtocolError(c, req.Id, "Server configuration not found", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
			return
		}

		var tools []mcp.ToolSchema
		var err error
		switch protoType {
		case cnst.BackendProtoHttp:
			// Get tools for HTTP backend
			tools, ok = s.state.prefixToTools[conn.Meta().Prefix]
			if !ok {
				tools = []mcp.ToolSchema{}
			}
		case cnst.BackendProtoStdio:
			transport, ok := s.state.prefixToTransport[conn.Meta().Prefix]
			if !ok {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}

			tools, err = transport.FetchTools(c.Request.Context())
			if err != nil {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}
		case cnst.BackendProtoSSE:
			transport, ok := s.state.prefixToTransport[conn.Meta().Prefix]
			if !ok {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}

			tools, err = transport.FetchTools(c.Request.Context())
			if err != nil {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}
		case cnst.BackendProtoStreamable:
			transport, ok := s.state.prefixToTransport[conn.Meta().Prefix]
			if !ok {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}

			tools, err = transport.FetchTools(c.Request.Context())
			if err != nil {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}
		default:
			s.sendProtocolError(c, req.Id, "Unsupported protocol type", http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
			return
		}

		s.sendSuccessResponse(c, conn, req, mcp.ListToolsResult{
			Tools: tools,
		}, false)
		return

	case mcp.ToolsCall:
		protoType, ok := s.state.prefixToProtoType[conn.Meta().Prefix]
		if !ok {
			s.sendProtocolError(c, req.Id, "Server configuration not found", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
			return
		}

		// Parse tool call parameters
		var params mcp.CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendProtocolError(c, req.Id, fmt.Sprintf("invalid tool call parameters: %v", err), http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
			return
		}

		var (
			result *mcp.CallToolResult
			err    error
		)
		switch protoType {
		case cnst.BackendProtoHttp:
			// For HTTP backend, use the existing HTTP tool invocation logic
			tool, exists := s.state.toolMap[params.Name]
			if !exists {
				s.sendProtocolError(c, req.Id, "Tool not found", http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
				return
			}

			// Convert arguments to map[string]any
			var args map[string]any
			if err := json.Unmarshal(params.Arguments, &args); err != nil {
				s.sendProtocolError(c, req.Id, "Invalid tool arguments", http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
				return
			}

			// Get server configuration
			serverCfg, ok := s.state.prefixToServerConfig[conn.Meta().Prefix]
			if !ok {
				s.sendProtocolError(c, req.Id, "Server config not found", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}

			// Execute the tool
			result, err = s.executeHTTPTool(conn, tool, args, c.Request, serverCfg.Config)
			if err != nil {
				s.logger.Error("failed to execute tool", zap.Error(err))
				s.sendToolExecutionError(c, conn, req, err, false)
				return
			}
		case cnst.BackendProtoStdio:
			transport, ok := s.state.prefixToTransport[conn.Meta().Prefix]
			if !ok {
				errMsg := "Server configuration not found"
				s.sendProtocolError(c, req.Id, errMsg, http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
				return
			}

			result, err = transport.CallTool(c.Request.Context(), params, mergeRequestInfo(conn.Meta().Request, c.Request))
			if err != nil {
				s.sendToolExecutionError(c, conn, req, err, true)
				return
			}
		case cnst.BackendProtoSSE:
			transport, ok := s.state.prefixToTransport[conn.Meta().Prefix]
			if !ok {
				errMsg := "Server configuration not found"
				s.sendProtocolError(c, req.Id, errMsg, http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
				return
			}

			result, err = transport.CallTool(c.Request.Context(), params, mergeRequestInfo(conn.Meta().Request, c.Request))
			if err != nil {
				s.sendToolExecutionError(c, conn, req, err, true)
				return
			}
		case cnst.BackendProtoStreamable:
			transport, ok := s.state.prefixToTransport[conn.Meta().Prefix]
			if !ok {
				errMsg := "Server configuration not found"
				s.sendProtocolError(c, req.Id, errMsg, http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
				return
			}

			result, err = transport.CallTool(c.Request.Context(), params, mergeRequestInfo(conn.Meta().Request, c.Request))
			if err != nil {
				s.sendToolExecutionError(c, conn, req, err, true)
				return
			}

		default:
			s.sendProtocolError(c, req.Id, "Unsupported protocol type", http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
			return
		}

		s.sendSuccessResponse(c, conn, req, result, false)
		return

	default:
		s.sendProtocolError(c, req.Id, "Method not found", http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
		return
	}
}

func (s *Server) getSession(c *gin.Context) session.Connection {
	sessionID := c.GetHeader(mcp.HeaderMcpSessionID)
	if sessionID == "" {
		s.sendProtocolError(c, nil, "Bad Request: Mcp-Session-Id header is required", http.StatusBadRequest, mcp.ErrorCodeConnectionClosed)
		return nil
	}
	conn, err := s.sessions.Get(c.Request.Context(), sessionID)
	if err != nil {
		s.sendProtocolError(c, nil, "Session not found", http.StatusNotFound, mcp.ErrorCodeRequestTimeout)
		return nil
	}
	return conn
}

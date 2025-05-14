package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/core/mcpproxy"

	"go.uber.org/zap"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// handleSSE handles SSE connections
func (s *Server) handleSSE(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")

	// Get the prefix from the request path
	prefix := strings.TrimSuffix(c.Request.URL.Path, "/sse")
	if prefix == "" {
		prefix = "/"
	}

	sessionID := uuid.New().String()
	meta := &session.Meta{
		ID:        sessionID,
		CreatedAt: time.Now(),
		Prefix:    prefix,
		Type:      "sse",
		Extra:     nil,
	}

	conn, err := s.sessions.Register(c.Request.Context(), meta)
	if err != nil {
		s.sendProtocolError(c, sessionID, "Failed to create SSE connection", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
		return
	}

	// Send the initial endpoint event
	_, err = fmt.Fprintf(c.Writer, "event: endpoint\ndata: %s\n\n",
		fmt.Sprintf("%s/message?sessionId=%s", strings.TrimSuffix(c.Request.URL.Path, "/sse"), meta.ID))
	if err != nil {
		s.sendProtocolError(c, sessionID, "Failed to initialize SSE connection", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
		return
	}
	c.Writer.Flush()

	// Main event loop
	for {
		select {
		case event := <-conn.EventQueue():
			switch event.Event {
			case "message":
				_, err = fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", event.Data)
				if err != nil {
					s.logger.Error("failed to send SSE message", zap.Error(err))
				}
			default:
				_, _ = fmt.Fprint(c.Writer, event)
			}
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		case <-s.shutdownCh:
			return
		}
	}
}

// sendErrorResponse sends an error response through SSE channel and returns Accepted status
func (s *Server) sendErrorResponse(c *gin.Context, conn session.Connection, req mcp.JSONRPCRequest, errorMsg string) {
	response := mcp.JSONRPCErrorSchema{
		JSONRPCBaseResult: mcp.JSONRPCBaseResult{
			JSONRPC: mcp.JSPNRPCVersion,
			ID:      req.Id,
		},
		Error: mcp.JSONRPCError{
			Code:    mcp.ErrorCodeInternalError,
			Message: errorMsg,
		},
	}
	eventData, err := json.Marshal(response)
	if err != nil {
		c.String(http.StatusAccepted, mcp.Accepted)
		return
	}
	err = conn.Send(c.Request.Context(), &session.Message{
		Event: "message",
		Data:  eventData,
	})
	if err != nil {
		c.String(http.StatusAccepted, mcp.Accepted)
		return
	}
	c.String(http.StatusAccepted, mcp.Accepted)
}

// handleMessage processes incoming JSON-RPC messages
func (s *Server) handleMessage(c *gin.Context) {
	s.logger.Debug("Received message", zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path))

	// Get the session ID from the query parameter
	sessionId := c.Query("sessionId")
	if sessionId == "" {
		c.String(http.StatusNotFound, "Missing sessionId parameter")
		s.sendProtocolError(c, nil, "Missing sessionId parameter", http.StatusBadRequest, mcp.ErrorCodeInvalidRequest)
		return
	}

	conn, err := s.sessions.Get(c.Request.Context(), sessionId)
	if err != nil {
		c.String(http.StatusNotFound, "Session not found")
		return
	}
	s.handlePostMessage(c, conn)
}

func (s *Server) handlePostMessage(c *gin.Context, conn session.Connection) {
	if conn == nil {
		c.String(http.StatusInternalServerError, "SSE connection not established")
		return
	}

	// Validate Content-Type header
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		c.String(http.StatusNotAcceptable, "Unsupported Media Type: Content-Type must be application/json")
		return
	}

	// TODO: support auth

	// Parse the JSON-RPC message
	var req mcp.JSONRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "Invalid message")
		return
	}

	switch req.Method {
	case mcp.NotificationInitialized:
		s.sendAcceptedResponse(c)
	case mcp.Initialize:
		var params mcp.InitializeRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendProtocolError(c, req.Id, "Invalid initialize parameters", http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
			return
		}

		result := mcp.InitializedResult{
			ProtocolVersion: mcp.LatestProtocolVersion,
			ServerInfo: mcp.ImplementationSchema{
				Name:    "mcp-gateway",
				Version: "0.1.0",
			},
			Capabilities: mcp.ServerCapabilitiesSchema{
				Tools: mcp.ToolsCapabilitySchema{
					ListChanged: true,
				},
			},
		}
		s.sendSuccessResponse(c, conn, req, result, true)
	case mcp.Ping:
		// Handle ping request with an empty response
		s.sendSuccessResponse(c, conn, req, struct{}{}, true)
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
			tools, err = s.fetchHTTPToolList(conn)
			if err != nil {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}
		case cnst.BackendProtoStdio:
			mcpProxyCfg, ok := s.state.prefixToMCPServerConfig[conn.Meta().Prefix]
			if !ok {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}

			tools, err = mcpproxy.FetchStdioToolList(c.Request.Context(), mcpProxyCfg)
			if err != nil {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}
		case cnst.BackendProtoSSE:
			mcpProxyCfg, ok := s.state.prefixToMCPServerConfig[conn.Meta().Prefix]
			if !ok {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}

			tools, err = mcpproxy.FetchSSEToolList(c.Request.Context(), mcpProxyCfg)
			if err != nil {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}
		case cnst.BackendProtoStreamable:
			mcpProxyCfg, ok := s.state.prefixToMCPServerConfig[conn.Meta().Prefix]
			if !ok {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}

			tools, err = mcpproxy.FetchStreamableToolList(c.Request.Context(), mcpProxyCfg)
			if err != nil {
				s.sendProtocolError(c, req.Id, "Failed to fetch tools", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
				return
			}
		default:
			s.sendProtocolError(c, req.Id, "Unsupported protocol type", http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
			return
		}

		toolSchemas := make([]mcp.ToolSchema, len(tools))
		for i, tool := range tools {
			toolSchemas[i] = mcp.ToolSchema{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			}
		}

		result := mcp.ListToolsResult{
			Tools: toolSchemas,
		}
		s.sendSuccessResponse(c, conn, req, result, true)
	case mcp.ToolsCall:
		protoType, ok := s.state.prefixToProtoType[conn.Meta().Prefix]
		if !ok {
			s.sendProtocolError(c, req.Id, "Server configuration not found", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
			return
		}

		// Execute the tool and return the result
		var params mcp.CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendProtocolError(c, req.Id, "Invalid tool call parameters", http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
			return
		}

		var (
			result *mcp.CallToolResult
			err    error
		)
		switch protoType {
		case cnst.BackendProtoHttp:
			result = s.invokeHTTPTool(c, req, conn, params)
		case cnst.BackendProtoStdio:
			mcpProxyCfg, ok := s.state.prefixToMCPServerConfig[conn.Meta().Prefix]
			if !ok {
				errMsg := "Server configuration not found"
				s.sendProtocolError(c, req.Id, errMsg, http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
				return
			}
			result, err = mcpproxy.InvokeStdioTool(c, mcpProxyCfg, params)
			if err != nil {
				s.sendToolExecutionError(c, conn, req, err, true)
				return
			}
		case cnst.BackendProtoSSE:
			mcpProxyCfg, ok := s.state.prefixToMCPServerConfig[conn.Meta().Prefix]
			if !ok {
				errMsg := "Server configuration not found"
				s.sendProtocolError(c, req.Id, errMsg, http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
				return
			}
			result, err = mcpproxy.InvokeSSETool(c, mcpProxyCfg, params)
			if err != nil {
				s.sendToolExecutionError(c, conn, req, err, true)
				return
			}
		case cnst.BackendProtoStreamable:
			mcpProxyCfg, ok := s.state.prefixToMCPServerConfig[conn.Meta().Prefix]
			if !ok {
				errMsg := "Server configuration not found"
				s.sendProtocolError(c, req.Id, errMsg, http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
				return
			}
			result, err = mcpproxy.InvokeStreamableTool(c, mcpProxyCfg, params)
			if err != nil {
				s.sendToolExecutionError(c, conn, req, err, true)
				return
			}
		default:
			s.sendProtocolError(c, req.Id, "Unsupported protocol type", http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
			return
		}

		s.sendSuccessResponse(c, conn, req, result, true)
	default:
		s.sendProtocolError(c, req.Id, "Unknown method", http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
	}
}

package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"

	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// handleSSE handles SSE connections
func (s *Server) handleSSE(c *gin.Context) {
	w := c.Writer
	r := c.Request

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store sess"})
		return
	}

	// Send the initial endpoint event
	_, _ = fmt.Fprintf(w, "event: endpoint\ndata: %s\r\n\r\n",
		fmt.Sprintf("%s/message?sessionId=%s", c.Request.URL.Path[:len(c.Request.URL.Path)-4], meta.ID))
	flusher.Flush()

	// Main event loop
	for {
		select {
		case event := <-conn.EventQueue():
			switch event.Event {
			case "message":
				_, _ = fmt.Fprint(w, fmt.Sprintf("event: message\ndata: %s\n\n", event.Data))
			}
			_, _ = fmt.Fprint(w, event)
			flusher.Flush()
		case <-r.Context().Done():
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
	// Parse the JSON-RPC message
	var req mcp.JSONRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendProtocolError(c, nil, mcp.ErrorCodeParseError, "Invalid JSON-RPC request", http.StatusBadRequest)
		return
	}

	// Get the session ID from the query parameter
	sessionId := c.Query("sessionId")
	if sessionId == "" {
		s.sendProtocolError(c, req.Id, mcp.ErrorCodeInvalidRequest, "Missing session ID", http.StatusBadRequest)
		return
	}

	conn, err := s.sessions.Get(c.Request.Context(), sessionId)
	if err != nil {
		s.sendProtocolError(c, req.Id, mcp.ErrorCodeInvalidRequest, "Invalid or expired session", http.StatusBadRequest)
		return
	}

	switch req.Method {
	case mcp.NotificationInitialized:
		// Do nothing, just acknowledge
		s.sendAcceptedResponse(c)
	case mcp.Initialize:
		var params mcp.InitializeRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeInvalidParams, "Invalid initialize parameters", http.StatusBadRequest)
			return
		}

		result := mcp.InitializedResult{
			ProtocolVersion: mcp.LatestProtocolVersion,
			ServerInfo: mcp.ImplementationSchema{
				Name:    "mcp-gateway",
				Version: "0.1.0",
			},
		}
		s.sendSuccessResponse(c, conn, req, result, true)
	case mcp.ToolsList:
		// Get tools for this prefix
		tools, ok := s.prefixToTools[conn.Meta().Prefix]
		if !ok {
			tools = []mcp.ToolSchema{} // Return empty list if prefix not found
		}

		result := mcp.ListToolsResult{
			Tools: tools,
		}
		s.sendSuccessResponse(c, conn, req, result, true)
	case mcp.ToolsCall:
		// Execute the tool and return the result
		var params mcp.CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeInvalidParams, "Invalid tool call parameters", http.StatusBadRequest)
			return
		}

		// Find the tool in the precomputed map
		tool, exists := s.toolMap[params.Name]
		if !exists {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeMethodNotFound, "Tool not found", http.StatusNotFound)
			return
		}

		// Convert arguments to map[string]any
		var args map[string]any
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeInvalidParams, "Invalid tool arguments", http.StatusBadRequest)
			return
		}

		serverCfg, ok := s.prefixToServerConfig[conn.Meta().Prefix]
		if !ok {
			s.sendProtocolError(c, req.Id, mcp.ErrorCodeInternalError, "Server configuration not found", http.StatusInternalServerError)
			return
		}

		// Execute the tool
		result, err := s.executeTool(tool, args, c.Request, serverCfg.Config)
		if err != nil {
			s.sendToolExecutionError(c, conn, req, err, true)
			return
		}

		// Send the result
		toolResult := mcp.CallToolResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: result,
				},
			},
			IsError: false,
		}
		s.sendSuccessResponse(c, conn, req, toolResult, true)
	default:
		s.sendProtocolError(c, req.Id, mcp.ErrorCodeMethodNotFound, "Unknown method", http.StatusNotFound)
	}
}

package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/core/mcpclient"
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
		}
		s.sendSuccessResponse(c, conn, req, result, true)
	case mcp.ToolsList:
		// Get the proto type for this prefix
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
			tools, err = s.fetchStdioToolList(c.Request.Context(), conn)
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
		// Get the proto type for this prefix
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

		var result *mcp.CallToolResult
		var err error
		switch protoType {
		case cnst.BackendProtoHttp:
			result, err = s.invokeHTTPTool(c, req, conn, params)
			if err != nil {
				// just return, the error already sent
				return
			}
		case cnst.BackendProtoStdio:
			result, err = s.invokeStdioTool(c, req, conn, params)
			if err != nil {
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

func (s *Server) fetchHTTPToolList(conn session.Connection) ([]mcp.ToolSchema, error) {
	// Get http tools for this prefix
	tools, ok := s.state.prefixToTools[conn.Meta().Prefix]
	if !ok {
		tools = []mcp.ToolSchema{} // Return empty list if prefix not found
	}

	return tools, nil
}

func (s *Server) fetchStdioToolList(ctx context.Context, conn session.Connection) ([]mcp.ToolSchema, error) {
	// Get stdio tools for this prefix
	stdioCfg, ok := s.state.prefixToStdioServerConfig[conn.Meta().Prefix]
	if !ok {
		return []mcp.ToolSchema{}, nil
	}

	stdioClientEnv := mcp.CoverToStdioClientEnv(stdioCfg.Env)
	stdioClient, err := mcpclient.NewStdioMCPClient(stdioCfg.Command, stdioClientEnv, stdioCfg.Args...)
	if err != nil {
		return []mcp.ToolSchema{}, err
	}
	defer stdioClient.Close()

	// initialize stdio client
	initRequest := mcp.InitializeRequestSchema{}
	var initParams mcp.InitializeRequestParams
	initParams.ProtocolVersion = mcp.LatestProtocolVersion
	initParams.ClientInfo = mcp.ImplementationSchema{
		Name:    "mcp-gateway",
		Version: "0.1.0",
	}
	paramsBytes, _ := json.Marshal(initParams)
	initRequest.Params = paramsBytes
	initRequest.Method = mcp.Initialize
	initRequest.JSONRPC = mcp.JSPNRPCVersion

	_, err = stdioClient.Initialize(ctx, initRequest)
	if err != nil {
		return []mcp.ToolSchema{}, err
	}

	// list tools
	listToolsResult, err := stdioClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return []mcp.ToolSchema{}, err
	}

	tools := make([]mcp.ToolSchema, len(listToolsResult.Tools))
	for i, schema := range listToolsResult.Tools {
		tools[i] = mcp.ToolSchema{
			Name:        schema.Name,
			Description: schema.Description,
			InputSchema: schema.InputSchema,
		}
	}

	return tools, nil
}

func (s *Server) invokeHTTPTool(c *gin.Context, req mcp.JSONRPCRequest, conn session.Connection, params mcp.CallToolParams) (*mcp.CallToolResult, error) {
	// Find the tool in the precomputed map
	tool, exists := s.state.toolMap[params.Name]
	if !exists {
		errMsg := "Tool not found"
		s.sendProtocolError(c, req.Id, errMsg, http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
		return nil, errors.New(errMsg)
	}

	// Convert arguments to map[string]any
	var args map[string]any
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		errMsg := "Invalid tool arguments"
		s.sendProtocolError(c, req.Id, errMsg, http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
		return nil, errors.New(errMsg)
	}

	// Get server configuration
	serverCfg, ok := s.state.prefixToServerConfig[conn.Meta().Prefix]
	if !ok {
		errMsg := "Server configuration not found"
		s.sendProtocolError(c, req.Id, errMsg, http.StatusInternalServerError, mcp.ErrorCodeInternalError)
		return nil, errors.New(errMsg)
	}

	// Execute the tool
	result, err := s.executeHTTPTool(tool, args, c.Request, serverCfg.Config)
	if err != nil {
		s.sendToolExecutionError(c, conn, req, err, true)
		return nil, err
	}

	return result, nil
}

func (s *Server) invokeStdioTool(c *gin.Context, req mcp.JSONRPCRequest, conn session.Connection, params mcp.CallToolParams) (*mcp.CallToolResult, error) {
	// Get stdio tools for this prefix
	stdioCfg, ok := s.state.prefixToStdioServerConfig[conn.Meta().Prefix]
	if !ok {
		errMsg := "Server configuration not found"
		s.sendProtocolError(c, req.Id, errMsg, http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
		return nil, errors.New(errMsg)
	}

	// Get server configuration
	serverCfg, ok := s.state.prefixToServerConfig[conn.Meta().Prefix]
	if !ok {
		errMsg := "Server configuration not found"
		s.sendProtocolError(c, req.Id, errMsg, http.StatusInternalServerError, mcp.ErrorCodeInternalError)
		return nil, errors.New(errMsg)
	}

	// Convert arguments to map[string]any
	var args map[string]any
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		errMsg := "Invalid tool arguments"
		s.sendProtocolError(c, req.Id, errMsg, http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
		return nil, errors.New(errMsg)
	}

	result, err := s.executeStdioTool(c, stdioCfg, args, c.Request, serverCfg.Config, params)
	if err != nil {
		s.sendToolExecutionError(c, conn, req, err, true)
		return nil, err
	}

	return result, nil
}

func (s *Server) executeStdioTool(
	c *gin.Context,
	tool *config.StdioServerConfig,
	args map[string]any,
	request *http.Request,
	serverCfg map[string]string,
	params mcp.CallToolParams,
) (*mcp.CallToolResult, error) {
	// Prepare template context
	tmplCtx, err := prepareTemplateContext(args, request, serverCfg)
	if err != nil {
		return nil, err
	}

	renderedClientEnv := make(map[string]string)
	for k, v := range tool.Env {
		rendered, err := renderTemplate(v, tmplCtx)
		if err != nil {
			return nil, err
		}
		renderedClientEnv[k] = rendered
	}

	toolCallRequestParams := make(map[string]interface{})
	if err := json.Unmarshal(params.Arguments, &toolCallRequestParams); err != nil {
		return nil, err
	}

	stdioClientEnv := mcp.CoverToStdioClientEnv(renderedClientEnv)
	stdioClient, err := mcpclient.NewStdioMCPClient(tool.Command, stdioClientEnv, tool.Args...)
	if err != nil {
		return nil, err
	}
	defer stdioClient.Close()

	// initialize stdio client
	initRequest := mcp.InitializeRequestSchema{}
	var initParams mcp.InitializeRequestParams
	initParams.ProtocolVersion = mcp.LatestProtocolVersion
	initParams.ClientInfo = mcp.ImplementationSchema{
		Name:    "mcp-gateway",
		Version: "0.1.0",
	}
	paramsBytes, _ := json.Marshal(initParams)
	initRequest.Params = paramsBytes
	initRequest.Method = mcp.Initialize
	initRequest.JSONRPC = mcp.JSPNRPCVersion

	_, err = stdioClient.Initialize(c.Request.Context(), initRequest)
	if err != nil {
		return nil, err
	}

	// call tool
	toolCallRequest := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
	}
	toolCallRequest.Params.Name = params.Name
	toolCallRequest.Params.Arguments = toolCallRequestParams

	result, err := stdioClient.CallTool(c.Request.Context(), toolCallRequest)
	if err != nil {
		return nil, err
	}

	return result, nil
}

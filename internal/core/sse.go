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
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
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
	stdioCfg, ok := s.state.prefixToMCPServerConfig[conn.Meta().Prefix]
	if !ok {
		return []mcp.ToolSchema{}, nil
	}

	// Create stdio transport with the command and arguments
	command := stdioCfg.Command
	args := stdioCfg.Args
	stdioClientEnv := mcp.CoverToStdioClientEnv(stdioCfg.Env)

	// Create mcp-go stdio transport
	stdioTransport := transport.NewStdio(command, stdioClientEnv, args...)

	// Start the transport
	if err := stdioTransport.Start(ctx); err != nil {
		return []mcp.ToolSchema{}, fmt.Errorf("failed to start stdio transport: %w", err)
	}

	// Create client with the transport
	c := client.NewClient(stdioTransport)
	defer c.Close()

	// Initialize the client
	initRequest := mcpgo.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcpgo.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcpgo.Implementation{
		Name:    "mcp-gateway",
		Version: "0.1.0",
	}

	_, err := c.Initialize(ctx, initRequest)
	if err != nil {
		return []mcp.ToolSchema{}, fmt.Errorf("failed to initialize: %w", err)
	}

	// List available tools
	toolsRequest := mcpgo.ListToolsRequest{}
	toolsResult, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		return []mcp.ToolSchema{}, fmt.Errorf("failed to list tools: %w", err)
	}

	// Convert from mcpgo.Tool to mcp.ToolSchema
	tools := make([]mcp.ToolSchema, len(toolsResult.Tools))
	for i, schema := range toolsResult.Tools {
		// Create local mcp package ToolInputSchema
		inputSchema := mcp.ToolInputSchema{
			Type:       "object",
			Properties: make(map[string]any),
		}

		// Convert mcpgo InputSchema to local mcp format
		rawSchema, err := json.Marshal(schema.InputSchema)
		if err == nil {
			// Parse schema properties
			var schemaMap map[string]interface{}
			if err := json.Unmarshal(rawSchema, &schemaMap); err == nil {
				if properties, ok := schemaMap["properties"].(map[string]interface{}); ok {
					inputSchema.Properties = properties
				}
				if typ, ok := schemaMap["type"].(string); ok {
					inputSchema.Type = typ
				}
				if required, ok := schemaMap["required"].([]interface{}); ok {
					reqStrings := make([]string, len(required))
					for j, r := range required {
						if rStr, ok := r.(string); ok {
							reqStrings[j] = rStr
						}
					}
					inputSchema.Required = reqStrings
				}
			}
		}

		tools[i] = mcp.ToolSchema{
			Name:        schema.Name,
			Description: schema.Description,
			InputSchema: inputSchema,
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
	stdioCfg, ok := s.state.prefixToMCPServerConfig[conn.Meta().Prefix]
	if !ok {
		errMsg := "Server configuration not found"
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

	result, err := s.executeStdioTool(c, &stdioCfg, args, c.Request, params)
	if err != nil {
		s.sendToolExecutionError(c, conn, req, err, true)
		return nil, err
	}

	return result, nil
}

func (s *Server) executeStdioTool(
	c *gin.Context,
	tool *config.MCPServerConfig,
	args map[string]any,
	request *http.Request,
	params mcp.CallToolParams,
) (*mcp.CallToolResult, error) {
	// Prepare template context
	tmplCtx, err := prepareTemplateContextForMCPBackend(args, request)
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

	// Use mcp-go client
	command := tool.Command
	cmdArgs := tool.Args
	stdioClientEnv := mcp.CoverToStdioClientEnv(renderedClientEnv)

	// Create stdio transport
	stdioTransport := transport.NewStdio(command, stdioClientEnv, cmdArgs...)

	// Start the transport
	if err := stdioTransport.Start(c.Request.Context()); err != nil {
		return nil, fmt.Errorf("failed to start stdio transport: %w", err)
	}

	// Create client
	mcpClient := client.NewClient(stdioTransport)
	defer mcpClient.Close()

	// Initialize client
	initRequest := mcpgo.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcpgo.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcpgo.Implementation{
		Name:    "mcp-gateway",
		Version: "0.1.0",
	}

	_, err = mcpClient.Initialize(c.Request.Context(), initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stdio client: %w", err)
	}

	// Call tool
	callRequest := mcpgo.CallToolRequest{}
	callRequest.Params.Name = params.Name

	// Convert parameters to mcp-go format
	callRequest.Params.Arguments = toolCallRequestParams

	mcpgoResult, err := mcpClient.CallTool(c.Request.Context(), callRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	// Convert mcp-go result to local mcp format
	result := &mcp.CallToolResult{
		IsError: mcpgoResult.IsError,
	}

	// Process content items
	if len(mcpgoResult.Content) > 0 {
		var validContents []mcp.Content

		for _, content := range mcpgoResult.Content {
			// Skip null content
			if content == nil {
				continue
			}

			// Try to get content type
			contentType := ""
			switch c := content.(type) {
			case *mcpgo.TextContent:
				contentType = "text"
				validContents = append(validContents, &mcp.TextContent{
					Type: "text",
					Text: c.Text,
				})
			case *mcpgo.ImageContent:
				contentType = "image"
				validContents = append(validContents, &mcp.ImageContent{
					Type:     "image",
					Data:     c.Data,
					MimeType: c.MIMEType,
				})
			case *mcpgo.AudioContent:
				contentType = "audio"
				validContents = append(validContents, &mcp.AudioContent{
					Type:     "audio",
					Data:     c.Data,
					MimeType: c.MIMEType,
				})
			default:
				// Try to parse from raw content
				rawContent, err := json.Marshal(content)
				if err == nil {
					var contentMap map[string]interface{}
					if json.Unmarshal(rawContent, &contentMap) == nil {
						if typ, ok := contentMap["type"].(string); ok {
							contentType = typ

							switch contentType {
							case "text":
								if text, ok := contentMap["text"].(string); ok {
									validContents = append(validContents, &mcp.TextContent{
										Type: "text",
										Text: text,
									})
								}
							case "image":
								data, _ := contentMap["data"].(string)
								mimeType, _ := contentMap["mimeType"].(string)
								validContents = append(validContents, &mcp.ImageContent{
									Type:     "image",
									Data:     data,
									MimeType: mimeType,
								})
							case "audio":
								data, _ := contentMap["data"].(string)
								mimeType, _ := contentMap["mimeType"].(string)
								validContents = append(validContents, &mcp.AudioContent{
									Type:     "audio",
									Data:     data,
									MimeType: mimeType,
								})
							}
						}
					}
				}
			}
		}

		if len(validContents) > 0 {
			result.Content = validContents
		}
	}

	return result, nil
}

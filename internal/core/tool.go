package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/template"
)

// prepareRequest prepares the HTTP request with templates and arguments
func prepareRequest(tool *config.ToolConfig, tmplCtx *template.Context) (*http.Request, error) {
	// Process endpoint template
	endpoint, err := template.RenderTemplate(tool.Endpoint, tmplCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to render endpoint template: %w", err)
	}

	// Process request body template
	var reqBody io.Reader
	if tool.RequestBody != "" {
		rendered, err := template.RenderTemplate(tool.RequestBody, tmplCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render request body template: %w", err)
		}
		reqBody = strings.NewReader(rendered)
	}

	req, err := http.NewRequest(tool.Method, endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Process header templates
	for k, v := range tool.Headers {
		rendered, err := template.RenderTemplate(v, tmplCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render header template: %w", err)
		}
		req.Header.Set(k, rendered)
	}

	return req, nil
}

// processArguments processes tool arguments and adds them to the request
func processArguments(req *http.Request, tool *config.ToolConfig, args map[string]any) {
	for _, arg := range tool.Args {
		value := fmt.Sprint(args[arg.Name])
		switch strings.ToLower(arg.Position) {
		case "header":
			req.Header.Set(arg.Name, value)
		case "query":
			q := req.URL.Query()
			q.Add(arg.Name, value)
			req.URL.RawQuery = q.Encode()
		case "form-data":
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)

			if err := writer.WriteField(arg.Name, value); err != nil {
				continue
			}

			if err := writer.Close(); err != nil {
				continue
			}

			req.Body = io.NopCloser(&b)
			req.Header.Set("Content-Type", writer.FormDataContentType())
		}
	}
}

// preprocessResponseData processes response data to handle []any type
func preprocessResponseData(data map[string]any) map[string]any {
	processed := make(map[string]any)

	for k, v := range data {
		switch val := v.(type) {
		case []any:
			ss, _ := json.Marshal(val)
			processed[k] = string(ss)
		case map[string]any:
			processed[k] = preprocessResponseData(val)
		default:
			processed[k] = v
		}
	}
	return processed
}

// executeHTTPTool executes a tool with the given arguments
func (s *Server) executeHTTPTool(conn session.Connection, tool *config.ToolConfig, args map[string]any, request *http.Request, serverCfg map[string]string) (*mcp.CallToolResult, error) {
	// Prepare template context
	tmplCtx, err := template.PrepareTemplateContext(conn.Meta().Request, args, request, serverCfg)
	if err != nil {
		return nil, err
	}

	// Prepare HTTP request
	req, err := prepareRequest(tool, tmplCtx)
	if err != nil {
		return nil, err
	}

	// Process arguments
	processArguments(req, tool, args)

	// Execute request
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	// Process response
	callToolResult, err := s.toolRespHandler.Handle(resp, tool, tmplCtx)
	if err != nil {
		return nil, err
	}
	return callToolResult, nil
}

func (s *Server) fetchHTTPToolList(conn session.Connection) ([]mcp.ToolSchema, error) {
	// Get http tools for this prefix
	tools, ok := s.state.prefixToTools[conn.Meta().Prefix]
	if !ok {
		tools = []mcp.ToolSchema{} // Return empty list if prefix not found
	}

	return tools, nil
}

func (s *Server) invokeHTTPTool(c *gin.Context, req mcp.JSONRPCRequest, conn session.Connection, params mcp.CallToolParams) *mcp.CallToolResult {
	// Find the tool in the precomputed map
	tool, exists := s.state.toolMap[params.Name]
	if !exists {
		errMsg := "Tool not found"
		s.sendProtocolError(c, req.Id, errMsg, http.StatusNotFound, mcp.ErrorCodeMethodNotFound)
		return nil
	}

	// Convert arguments to map[string]any
	var args map[string]any
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		errMsg := "Invalid tool arguments"
		s.sendProtocolError(c, req.Id, errMsg, http.StatusBadRequest, mcp.ErrorCodeInvalidParams)
		return nil
	}

	// Get server configuration
	serverCfg, ok := s.state.prefixToServerConfig[conn.Meta().Prefix]
	if !ok {
		errMsg := "Server configuration not found"
		s.sendProtocolError(c, req.Id, errMsg, http.StatusInternalServerError, mcp.ErrorCodeInternalError)
		return nil
	}

	// Execute the tool
	result, err := s.executeHTTPTool(conn, tool, args, c.Request, serverCfg.Config)
	if err != nil {
		s.sendToolExecutionError(c, conn, req, err, true)
		return nil
	}

	return result
}

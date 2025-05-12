package core

import (
	"encoding/json"
	"fmt"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"
	"io"
	"net/http"
	"strings"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/template"
)

// renderTemplate renders a template with the given context
func renderTemplate(tmpl string, ctx *template.Context) (string, error) {
	renderer := template.NewRenderer()
	return renderer.Render(tmpl, ctx)
}

// prepareTemplateContext prepares the template context with request and config data
func prepareTemplateContext(args map[string]any, request *http.Request, serverCfg map[string]string) (*template.Context, error) {
	tmplCtx := template.NewContext()
	tmplCtx.Args = preprocessArgs(args)

	// Process server config templates
	for k, v := range serverCfg {
		rendered, err := renderTemplate(v, tmplCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render config template: %w", err)
		}
		serverCfg[k] = rendered
	}
	tmplCtx.Config = serverCfg

	// Process request headers
	for k, v := range request.Header {
		if len(v) > 0 {
			tmplCtx.Request.Headers[k] = v[0]
		}
	}

	return tmplCtx, nil
}

// prepareRequest prepares the HTTP request with templates and arguments
func prepareRequest(tool *config.ToolConfig, tmplCtx *template.Context) (*http.Request, error) {
	// Process endpoint template
	endpoint, err := renderTemplate(tool.Endpoint, tmplCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to render endpoint template: %w", err)
	}

	// Process request body template
	var reqBody io.Reader
	if tool.RequestBody != "" {
		rendered, err := renderTemplate(tool.RequestBody, tmplCtx)
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
		rendered, err := renderTemplate(v, tmplCtx)
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

// processResponse processes the HTTP response and applies response template if needed
func processResponse(resp *http.Response, tool *config.ToolConfig, tmplCtx *template.Context) (string, error) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	if tool.ResponseBody == "" {
		return string(respBody), nil
	}

	var respData map[string]any
	if err := json.Unmarshal(respBody, &respData); err != nil {
		// 非JSON格式的响应，忽略解析错误
	}

	// Preprocess response data to handle []any type
	respData = preprocessResponseData(respData)
	tmplCtx.Response.Data = respData
	tmplCtx.Response.Body = string(respBody)

	rendered, err := renderTemplate(tool.ResponseBody, tmplCtx)
	if err != nil {
		return "", fmt.Errorf("failed to render response body template: %w", err)
	}
	return rendered, nil
}

// executeTool executes a tool with the given arguments
func (s *Server) executeTool(tool *config.ToolConfig, args map[string]any, request *http.Request, serverCfg map[string]string) (*mcp.CallToolResult, error) {
	// Prepare template context
	tmplCtx, err := prepareTemplateContext(args, request, serverCfg)
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
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	// Process response
	callToolResult, err := ResponseHandlerChain.Handle(resp, tool, tmplCtx)
	if err != nil {
		return nil, err
	}
	return callToolResult, nil
}

func preprocessArgs(args map[string]any) map[string]any {
	processed := make(map[string]any)

	for k, v := range args {
		switch val := v.(type) {
		case []any:
			ss, _ := json.Marshal(val)
			processed[k] = string(ss)
		default:
			processed[k] = v
		}
	}
	return processed
}

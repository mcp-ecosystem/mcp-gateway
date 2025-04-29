package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"io"
	"net/http"
	"strings"
	texttemplate "text/template"

	"github.com/mcp-ecosystem/mcp-gateway/internal/template"
)

// executeTool executes a tool with the given arguments
func (s *Server) executeTool(tool *config.ToolConfig, args map[string]any, request *http.Request, serverCfg map[string]string) (string, error) {
	client := &http.Client{}

	tmplCtx := template.NewContext()
	tmplCtx.Args = args

	tmplCtx.Config = serverCfg

	for k, v := range request.Header {
		if len(v) > 0 {
			tmplCtx.Request.Headers[k] = v[0]
		}
	}

	endpointTmpl, err := texttemplate.New("endpoint").Parse(tool.Endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse endpoint template: %w", err)
	}

	var endpointBuf bytes.Buffer
	if err := endpointTmpl.Execute(&endpointBuf, tmplCtx); err != nil {
		return "", fmt.Errorf("failed to execute endpoint template: %w", err)
	}
	endpoint := endpointBuf.String()

	var reqBody io.Reader
	if tool.RequestBody != "" {
		bodyTmpl, err := texttemplate.New("body").Parse(tool.RequestBody)
		if err != nil {
			return "", fmt.Errorf("failed to parse request body template: %w", err)
		}

		var bodyBuf bytes.Buffer
		if err := bodyTmpl.Execute(&bodyBuf, tmplCtx); err != nil {
			return "", fmt.Errorf("failed to execute request body template: %w", err)
		}
		reqBody = &bodyBuf
	}

	req, err := http.NewRequest(tool.Method, endpoint, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range tool.Headers {
		headerTmpl, err := texttemplate.New("header").Parse(v)
		if err != nil {
			return "", fmt.Errorf("failed to parse header template: %w", err)
		}

		var headerBuf bytes.Buffer
		if err := headerTmpl.Execute(&headerBuf, tmplCtx); err != nil {
			return "", fmt.Errorf("failed to execute header template: %w", err)
		}
		req.Header.Set(k, headerBuf.String())
	}

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

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if tool.ResponseBody != "" {
		respTmpl, err := texttemplate.New("response").Parse(tool.ResponseBody)
		if err != nil {
			return "", fmt.Errorf("failed to parse response body template: %w", err)
		}

		var respData map[string]any
		if err := json.Unmarshal(respBody, &respData); err != nil {
			// TODO: ignore the error for now, in case the response is not JSON
		}

		tmplCtx.Response.Data = respData
		tmplCtx.Response.Body = string(respBody)
		var respBuf bytes.Buffer
		if err := respTmpl.Execute(&respBuf, tmplCtx); err != nil {
			return "", fmt.Errorf("failed to execute response body template: %w", err)
		}
		return respBuf.String(), nil
	}

	return string(respBody), nil
}

package mcpproxy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mcp-ecosystem/mcp-gateway/internal/template"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/utils"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/version"
)

// StdioTransport implements Transport using standard input/output
type StdioTransport struct {
	client *client.Client
	cfg    config.MCPServerConfig
}

var _ Transport = (*StdioTransport)(nil)

func (t *StdioTransport) Start(ctx context.Context, tmplCtx *template.Context) error {
	if t.IsStarted() {
		return nil
	}

	renderedClientEnv := make(map[string]string)
	for k, v := range t.cfg.Env {
		rendered, err := template.RenderTemplate(v, tmplCtx)
		if err != nil {
			return fmt.Errorf("failed to render env template: %w", err)
		}
		renderedClientEnv[k] = rendered
	}

	// Create stdio transport
	stdioTransport := transport.NewStdio(
		t.cfg.Command,
		utils.MapToEnvList(renderedClientEnv),
		t.cfg.Args...,
	)
	fmt.Println("debug:", utils.MapToEnvList(renderedClientEnv), t.cfg.Command, t.cfg.Args)

	// Start the transport
	if err := stdioTransport.Start(ctx); err != nil {
		return fmt.Errorf("failed to start stdio transport: %w", err)
	}

	// Create client with the transport
	c := client.NewClient(stdioTransport)

	// Initialize the client
	initRequest := mcpgo.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcpgo.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcpgo.Implementation{
		Name:    "mcp-gateway",
		Version: version.Get(),
	}

	_, err := c.Initialize(ctx, initRequest)
	if err != nil {
		_ = stdioTransport.Close()
		return fmt.Errorf("failed to initialize stdio client: %w", err)
	}

	t.client = c
	return nil
}

func (t *StdioTransport) Stop(_ context.Context) error {
	if !t.IsStarted() {
		return nil
	}

	if t.client != nil {
		return t.client.Close()
	}

	return nil
}

func (t *StdioTransport) IsStarted() bool {
	return t.client != nil
}

// FetchToolList implements Transport.FetchToolList
func (t *StdioTransport) FetchToolList(ctx context.Context, _ session.Connection) ([]mcp.ToolSchema, error) {
	if !t.IsStarted() {
		if err := t.Start(ctx, template.NewContext()); err != nil {
			return nil, err
		}
	}
	defer func() {
		if t.cfg.Policy == cnst.PolicyOnDemand {
			_ = t.Stop(ctx)
		}
	}()

	// List available tools
	toolsResult, err := t.client.ListTools(ctx, mcpgo.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
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

// InvokeTool implements Transport.InvokeTool
func (t *StdioTransport) InvokeTool(ctx *gin.Context, conn session.Connection, params mcp.CallToolParams) (*mcp.CallToolResult, error) {
	if !t.IsStarted() {
		var args map[string]any
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid tool arguments: %w", err)
		}
		tmplCtx, err := template.PrepareTemplateContext(conn.Meta().Request, args, ctx.Request, nil)
		if err != nil {
			return nil, err
		}
		if err := t.Start(ctx.Request.Context(), tmplCtx); err != nil {
			return nil, err
		}
	}
	defer func() {
		if t.cfg.Policy == cnst.PolicyOnDemand {
			_ = t.Stop(ctx)
		}
	}()

	// Convert arguments to map[string]any
	var args map[string]any
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}

	toolCallRequestParams := make(map[string]interface{})
	if err := json.Unmarshal(params.Arguments, &toolCallRequestParams); err != nil {
		return nil, err
	}

	// Call tool
	callRequest := mcpgo.CallToolRequest{}
	callRequest.Params.Name = params.Name
	callRequest.Params.Arguments = toolCallRequestParams

	mcpgoResult, err := t.client.CallTool(ctx.Request.Context(), callRequest)
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

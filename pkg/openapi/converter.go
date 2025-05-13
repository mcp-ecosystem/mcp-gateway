package openapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
)

// Converter handles the conversion from OpenAPI to MCP configuration
type Converter struct {
	// Add any necessary fields here
}

// NewConverter creates a new Converter instance
func NewConverter() *Converter {
	return &Converter{}
}

// Convert converts OpenAPI specification to MCP configuration
func (c *Converter) Convert(specData []byte) (*config.MCPConfig, error) {
	// Parse OpenAPI specification
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(specData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI specification: %w", err)
	}

	// Validate the OpenAPI document
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI specification: %w", err)
	}

	// Create base MCP configuration
	mcpConfig := &config.MCPConfig{
		Name:      doc.Info.Title,
		Tenant:    "/default", // Default tenant prefix
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Routers:   make([]config.RouterConfig, 0),
		Servers:   make([]config.ServerConfig, 0),
		Tools:     make([]config.ToolConfig, 0),
	}

	// Create server configuration
	server := config.ServerConfig{
		Name:         doc.Info.Title,
		Description:  doc.Info.Description,
		Config:       make(map[string]string),
		AllowedTools: make([]string, 0),
	}

	// Add server URL to config
	if len(doc.Servers) > 0 {
		server.Config["url"] = doc.Servers[0].URL
	}

	// Create a default router for the server
	router := config.RouterConfig{
		Server: doc.Info.Title,
		Prefix: "/mcp", // TODO: Get prefix from tenant
		CORS: &config.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "OPTIONS"},
			AllowHeaders:     []string{"Content-Type", "Authorization", "Mcp-Session-Id"},
			ExposeHeaders:    []string{"Mcp-Session-Id"},
			AllowCredentials: true,
		},
	}

	// Convert paths to tools
	for path, pathItem := range doc.Paths.Map() {
		// Create a tool for each HTTP method
		for method, operation := range pathItem.Operations() {
			if method == "options" {
				continue // Skip CORS options
			}

			// Skip if operation ID is empty
			if operation.OperationID == "" {
				// Generate operationId from method and path
				// Convert path to operationId format: /users/email/{email} -> users_email_argemail
				pathParts := strings.Split(strings.TrimPrefix(path, "/"), "/")
				for i, part := range pathParts {
					if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
						pathParts[i] = "arg" + strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
					}
				}
				operation.OperationID = fmt.Sprintf("%s_%s", strings.ToLower(method), strings.Join(pathParts, "_"))
			}

			tool := config.ToolConfig{
				Name:         operation.OperationID,
				Description:  operation.Summary,
				Method:       method,
				Endpoint:     fmt.Sprintf("{{.Config.url}}%s", path),
				Headers:      make(map[string]string),
				Args:         make([]config.ArgConfig, 0),
				ResponseBody: "{{.Response.Body}}", // Use passthrough for response
			}

			// Add default headers
			tool.Headers["Content-Type"] = "application/json"
			tool.Headers["Authorization"] = "{{.Request.Headers.Authorization}}"

			// Add parameters
			var bodyArgs []config.ArgConfig
			var pathArgs []config.ArgConfig
			var queryArgs []config.ArgConfig

			// Handle path parameters
			for _, param := range operation.Parameters {
				arg := config.ArgConfig{
					Name:        param.Value.Name,
					Position:    param.Value.In,
					Required:    param.Value.Required,
					Type:        "string", // Default to string type
					Description: param.Value.Description,
				}

				// Get schema type if available
				if param.Value.Schema != nil && param.Value.Schema.Value != nil {
					if param.Value.Schema.Value.Type != nil {
						types := param.Value.Schema.Value.Type.Slice()
						if len(types) > 0 {
							arg.Type = types[0]
						}
					}
					if param.Value.Schema.Value.Default != nil {
						arg.Default = fmt.Sprintf("%v", param.Value.Schema.Value.Default)
					}
				}

				switch param.Value.In {
				case "path":
					// Path parameters are always required
					arg.Required = true
					pathArgs = append(pathArgs, arg)
					// Update endpoint with path parameters
					tool.Endpoint = strings.ReplaceAll(tool.Endpoint, fmt.Sprintf("{%s}", arg.Name), fmt.Sprintf("{{.Args.%s}}", arg.Name))
				case "query":
					queryArgs = append(queryArgs, arg)
				case "header":
					tool.Headers[arg.Name] = fmt.Sprintf("{{.Args.%s}}", arg.Name)
				}
			}

			// Handle request body
			if operation.RequestBody != nil {
				requestBodyRequired := operation.RequestBody.Value.Required
				for contentType, mediaType := range operation.RequestBody.Value.Content {
					if contentType == "application/json" {
						tool.RequestBody = contentType
						// Add request body parameters
						if mediaType.Schema != nil {
							schema := mediaType.Schema.Value
							// Handle schema reference
							if mediaType.Schema.Ref != "" {
								refName := strings.TrimPrefix(mediaType.Schema.Ref, "#/components/schemas/")
								if refSchema, ok := doc.Components.Schemas[refName]; ok {
									schema = refSchema.Value
								}
							}

							if schema.Properties != nil {
								for name, prop := range schema.Properties {
									// Skip response-only fields
									if strings.HasPrefix(name, "response") || name == "id" || name == "createdAt" {
										continue
									}

									arg := config.ArgConfig{
										Name:        name,
										Position:    "body",
										Required:    requestBodyRequired || contains(schema.Required, name),
										Type:        "string", // Default to string type
										Description: prop.Value.Description,
									}

									if prop.Value != nil && prop.Value.Type != nil {
										types := prop.Value.Type.Slice()
										if len(types) > 0 {
											arg.Type = types[0]
										}
									}

									if prop.Value.Default != nil {
										arg.Default = fmt.Sprintf("%v", prop.Value.Default)
									}

									bodyArgs = append(bodyArgs, arg)
								}
							}
						}
					}
				}
			}

			// Combine all args
			tool.Args = append(tool.Args, pathArgs...)
			tool.Args = append(tool.Args, queryArgs...)
			tool.Args = append(tool.Args, bodyArgs...)

			// Build request body template if there are body args
			if len(bodyArgs) > 0 {
				var bodyTemplate strings.Builder
				bodyTemplate.WriteString("{\n")
				for i, arg := range bodyArgs {
					bodyTemplate.WriteString(fmt.Sprintf(`    "%s": "{{.Args.%s}}"`, arg.Name, arg.Name))
					if i < len(bodyArgs)-1 {
						bodyTemplate.WriteString(",\n")
					} else {
						bodyTemplate.WriteString("\n")
					}
				}
				bodyTemplate.WriteString("}")
				tool.RequestBody = bodyTemplate.String()
			}

			mcpConfig.Tools = append(mcpConfig.Tools, tool)
			server.AllowedTools = append(server.AllowedTools, tool.Name)
		}
	}

	mcpConfig.Servers = append(mcpConfig.Servers, server)
	mcpConfig.Routers = append(mcpConfig.Routers, router)

	return mcpConfig, nil
}

// ConvertFromJSON converts JSON OpenAPI specification to MCP configuration
func (c *Converter) ConvertFromJSON(jsonData []byte) (*config.MCPConfig, error) {
	return c.Convert(jsonData)
}

// ConvertFromYAML converts YAML OpenAPI specification to MCP configuration
func (c *Converter) ConvertFromYAML(yamlData []byte) (*config.MCPConfig, error) {
	return c.Convert(yamlData)
}

// contains checks if a string is in a slice
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

package swagger

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/ifuryst/lol"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
)

// Converter handles the conversion from Swagger 2.0 to MCP configuration
type Converter struct {
	// Add any necessary fields here
}

// NewConverter creates a new Converter instance
func NewConverter() *Converter {
	return &Converter{}
}

// Convert converts Swagger 2.0 specification to MCP configuration
func (c *Converter) Convert(specData []byte) (*config.MCPConfig, error) {
	// Parse Swagger specification
	doc, err := loads.Analyzed(specData, "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse Swagger specification: %w", err)
	}

	swaggerSpec := doc.Spec()

	rs := lol.RandomString(4)

	// Create base MCP configuration
	mcpConfig := &config.MCPConfig{
		Name:      swaggerSpec.Info.Title + "_" + rs,
		Tenant:    "/default", // Default tenant prefix
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Routers:   make([]config.RouterConfig, 0),
		Servers:   make([]config.ServerConfig, 0),
		Tools:     make([]config.ToolConfig, 0),
	}

	// Create server configuration
	server := config.ServerConfig{
		Name:         mcpConfig.Name,
		Description:  swaggerSpec.Info.Description,
		Config:       make(map[string]string),
		AllowedTools: make([]string, 0),
	}

	// Add server URL to config
	if swaggerSpec.Host != "" {
		scheme := "https"
		if len(swaggerSpec.Schemes) > 0 {
			scheme = swaggerSpec.Schemes[0]
		}
		basePath := swaggerSpec.BasePath
		if basePath == "" {
			basePath = "/"
		}
		server.Config["url"] = fmt.Sprintf("%s://%s%s", scheme, swaggerSpec.Host, basePath)
	}

	// Create a default router for the server
	router := config.RouterConfig{
		Server: mcpConfig.Name,
		Prefix: fmt.Sprintf("/mcp/%s", rs), // Generate a random prefix for each router
		CORS: &config.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "OPTIONS"},
			AllowHeaders:     []string{"Content-Type", "Authorization", "Mcp-Session-Id"},
			ExposeHeaders:    []string{"Mcp-Session-Id"},
			AllowCredentials: true,
		},
	}

	// Convert paths to tools
	for path, pathItem := range swaggerSpec.Paths.Paths {
		// Create a tool for each HTTP method
		operations := map[string]*spec.Operation{
			"get":     pathItem.Get,
			"post":    pathItem.Post,
			"put":     pathItem.Put,
			"delete":  pathItem.Delete,
			"patch":   pathItem.Patch,
			"head":    pathItem.Head,
			"options": pathItem.Options,
		}

		for method, operation := range operations {
			if operation == nil || method == "options" {
				continue // Skip empty operations or CORS options
			}

			// Skip if operation ID is empty
			operationID := operation.ID
			if operationID == "" {
				// Generate operationId from method and path
				// Convert path to operationId format: /users/email/{email} -> users_email_argemail
				pathParts := strings.Split(strings.TrimPrefix(path, "/"), "/")
				for i, part := range pathParts {
					if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
						pathParts[i] = "arg" + strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
					}
				}
				operationID = fmt.Sprintf("%s_%s", strings.ToLower(method), strings.Join(pathParts, "_"))
			}

			tool := config.ToolConfig{
				Name:         operationID,
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

			// Handle parameters
			for _, param := range operation.Parameters {
				arg := config.ArgConfig{
					Name:        param.Name,
					Position:    param.In,
					Required:    param.Required,
					Type:        "string", // Default to string type
					Description: param.Description,
				}

				// Get schema type if available
				switch param.Type {
				case "integer", "number", "boolean", "array", "object":
					arg.Type = param.Type
				default:
					arg.Type = "string"
				}

				if param.Default != nil {
					arg.Default = fmt.Sprintf("%v", param.Default)
				}

				switch param.In {
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
				case "body":
					// Handle body parameter (Swagger 2.0 specific)
					if param.Schema != nil {
						if param.Schema.Properties != nil {
							for name, prop := range param.Schema.Properties {
								// Skip response-only fields
								if strings.HasPrefix(name, "response") || name == "id" || name == "createdAt" {
									continue
								}

								bodyArg := config.ArgConfig{
									Name:        name,
									Position:    "body",
									Required:    param.Required || contains(param.Schema.Required, name),
									Type:        "string", // Default to string type
									Description: prop.Description,
								}

								if prop.Type != nil && len(prop.Type) > 0 {
									bodyArg.Type = prop.Type[0]
								}

								if prop.Default != nil {
									bodyArg.Default = fmt.Sprintf("%v", prop.Default)
								}

								bodyArgs = append(bodyArgs, bodyArg)
							}
						}
					} else {
						// Simple body parameter
						bodyArgs = append(bodyArgs, arg)
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

// ConvertFromJSON converts JSON Swagger specification to MCP configuration
func (c *Converter) ConvertFromJSON(jsonData []byte) (*config.MCPConfig, error) {
	return c.Convert(jsonData)
}

// ConvertFromYAML converts YAML Swagger specification to MCP configuration
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

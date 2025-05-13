package config

import (
	"time"

	"github.com/ifuryst/lol"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"
)

type (
	// MCPServer represents the MCP server data structure
	MCPServer struct {
		Name      string    `json:"name" yaml:"name" gorm:"primaryKey"`
		Content   MCPConfig `json:"content" yaml:"content" gorm:"type:json"`
		CreatedAt time.Time `json:"createdAt" yaml:"created_at"`
		UpdatedAt time.Time `json:"updatedAt" yaml:"updated_at"`
	}

	MCPConfig struct {
		Name        string            `yaml:"name" gorm:"primaryKey"`
		Tenant      string            `yaml:"tenant" gorm:"index"`
		CreatedAt   time.Time         `yaml:"created_at"`
		UpdatedAt   time.Time         `yaml:"updated_at"`
		ProtoType   cnst.ProtoType    `yaml:"protoType" gorm:"type:varchar(64)"`
		Routers     []RouterConfig    `yaml:"routers" gorm:"type:json"`
		Servers     []ServerConfig    `yaml:"servers" gorm:"type:json"`
		Tools       []ToolConfig      `yaml:"tools" gorm:"type:json"`
		StdioServer StdioServerConfig `yaml:"stdioServer" gorm:"type:json"`
		SSEServer   SSEServerConfig   `yaml:"sseServer" gorm:"type:json"`
	}

	RouterConfig struct {
		Server string      `yaml:"server"`
		Prefix string      `yaml:"prefix"`
		CORS   *CORSConfig `yaml:"cors,omitempty"`
	}

	CORSConfig struct {
		AllowOrigins     []string `yaml:"allowOrigins"`
		AllowMethods     []string `yaml:"allowMethods"`
		AllowHeaders     []string `yaml:"allowHeaders"`
		ExposeHeaders    []string `yaml:"exposeHeaders"`
		AllowCredentials bool     `yaml:"allowCredentials"`
	}

	ServerConfig struct {
		Name         string            `yaml:"name"`
		Namespace    string            `yaml:"namespace"`
		Description  string            `yaml:"description"`
		AllowedTools []string          `yaml:"allowedTools"`
		Config       map[string]string `yaml:"config,omitempty"`
	}

	ToolConfig struct {
		Name         string            `yaml:"name"`
		Description  string            `yaml:"description,omitempty"`
		Method       string            `yaml:"method"`
		Endpoint     string            `yaml:"endpoint"`
		Headers      map[string]string `yaml:"headers"`
		Args         []ArgConfig       `yaml:"args"`
		RequestBody  string            `yaml:"requestBody"`
		ResponseBody string            `yaml:"responseBody"`
		InputSchema  map[string]any    `yaml:"inputSchema,omitempty"`
	}

	StdioServerConfig struct {
		Name    string            `yaml:"name"`
		Command string            `yaml:"command"`
		Args    []string          `yaml:"args"`
		Env     map[string]string `yaml:"env"`
	}

	SSEServerConfig struct {
		Name string `yaml:"name"`
		URL  string `yaml:"url"`
	}

	ArgConfig struct {
		Name        string      `yaml:"name" json:"name"`
		Position    string      `yaml:"position" json:"position"` // header, query, path, body
		Required    bool        `yaml:"required" json:"required"`
		Type        string      `yaml:"type" json:"type"`
		Description string      `yaml:"description" json:"description"`
		Default     string      `yaml:"default" json:"default"`
		Items       ItemsConfig `yaml:"items,omitempty" json:"items,omitempty"`
	}

	ItemsConfig struct {
		Type string   `yaml:"type" json:"type"`
		Enum []string `yaml:"enum,omitempty" json:"enum,omitempty"`
	}
)

// ToToolSchema converts a ToolConfig to a ToolSchema
func (t *ToolConfig) ToToolSchema() mcp.ToolSchema {
	// Create properties map for input schema
	properties := make(map[string]any)
	required := make([]string, 0)
	for _, arg := range t.Args {
		property := map[string]any{
			"type":        arg.Type,
			"description": arg.Description,
		}

		if arg.Type == "array" {
			items := make(map[string]any)
			if len(arg.Items.Enum) > 0 {
				items["enum"] = lol.Union(arg.Items.Enum)
			} else {
				items["type"] = arg.Items.Type
			}
			property["items"] = items
		}

		properties[arg.Name] = property
		if arg.Required {
			required = append(required, arg.Name)
		}
	}

	// Merge with existing input schema if any
	if t.InputSchema != nil {
		for k, v := range t.InputSchema {
			properties[k] = v
		}
	}

	return mcp.ToolSchema{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: properties,
			Required:   required,
		},
	}
}

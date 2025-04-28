package config

import (
	"time"

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
		Name         string                `yaml:"name" gorm:"primaryKey"`
		CreatedAt    time.Time             `yaml:"created_at"`
		UpdatedAt    time.Time             `yaml:"updated_at"`
		ProtoType    cnst.ProtoType        `yaml:"proto_type" gorm:"type:varchar(64); required"`
		Routers      []RouterConfig        `yaml:"routers" gorm:"type:json"`
		Servers      []ServerConfig        `yaml:"servers" gorm:"type:json"`
		Tools        []ToolConfig          `yaml:"tools" gorm:"type:json"`
		StdioServers StdioMcpServerConfigs `yaml:"stdio_servers" gorm:"type:json"`
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

	ArgConfig struct {
		Name        string `yaml:"name"`
		Position    string `yaml:"position"` // header, query, path, body
		Required    bool   `yaml:"required"`
		Type        string `yaml:"type"`
		Description string `yaml:"description"`
		Default     string `yaml:"default"`
	}

	StdioMcpServerConfigs map[string]map[string]StdioServerConfig

	StdioServerConfig struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
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
			"required":    arg.Required,
		}
		if arg.Description != "" {
			property["title"] = arg.Description
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

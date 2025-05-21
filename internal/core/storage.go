package core

import (
	"context"
	"time"
)

// Storage defines the interface for storing runtimeUnit data
type Storage interface {
	// SaveTool saves a tool configuration
	SaveTool(ctx context.Context, tool *Tool) error

	// GetTool retrieves a tool configuration
	GetTool(ctx context.Context, name string) (*Tool, error)

	// ListTools lists all tool configurations
	ListTools(ctx context.Context) ([]*Tool, error)

	// DeleteTool deletes a tool configuration
	DeleteTool(ctx context.Context, name string) error

	// SaveServer saves a server configuration
	SaveServer(ctx context.Context, server *StoredServer) error

	// GetServer retrieves a server configuration
	GetServer(ctx context.Context, name string) (*StoredServer, error)

	// ListServers lists all server configurations
	ListServers(ctx context.Context) ([]*StoredServer, error)

	// DeleteServer deletes a server configuration
	DeleteServer(ctx context.Context, name string) error
}

// Tool represents a tool in storage
type Tool struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Method       string            `json:"method"`
	Endpoint     string            `json:"endpoint"`
	Headers      map[string]string `json:"headers"`
	Args         []Arg             `json:"args"`
	RequestBody  string            `json:"requestBody"`
	ResponseBody string            `json:"responseBody"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// StoredServer represents a server in storage
type StoredServer struct {
	Name           string    `json:"name"`
	Namespace      string    `json:"namespace"`
	Description    string    `json:"description"`
	Auth           Auth      `json:"auth"`
	AllowedTools   []string  `json:"allowedTools"`
	AllowedOrigins []string  `json:"allowedOrigins"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Arg represents a tool argument
type Arg struct {
	Name        string `json:"name"`
	Position    string `json:"position"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     string `json:"default"`
}

// Auth represents authentication configuration
type Auth struct {
	Mode   string `json:"mode"`
	Header string `json:"header"`
	ArgKey string `json:"argKey"`
}

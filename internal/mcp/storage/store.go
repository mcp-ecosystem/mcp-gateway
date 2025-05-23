package storage

import (
	"context"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
)

// Store defines the interface for MCP configuration storage
type Store interface {
	// Create creates a new MCP configuration
	Create(ctx context.Context, cfg *config.MCPConfig) error

	// Get gets an MCP configuration by name
	Get(ctx context.Context, name string) (*config.MCPConfig, error)

	// List lists all MCP configurations
	List(ctx context.Context) ([]*config.MCPConfig, error)

	// Update updates an existing MCP configuration
	Update(ctx context.Context, cfg *config.MCPConfig) error

	// Delete deletes an MCP configuration
	Delete(ctx context.Context, name string) error

	// GetVersion gets a specific version of the configuration
	GetVersion(ctx context.Context, name string, version int) (*config.MCPConfigVersion, error)

	// ListVersions lists all versions of a configuration
	ListVersions(ctx context.Context, name string) ([]*config.MCPConfigVersion, error)

	// DeleteVersion deletes a specific version
	DeleteVersion(ctx context.Context, name string, version int) error

	// SetActiveVersion sets a specific version as the active version
	SetActiveVersion(ctx context.Context, name string, version int) error
}

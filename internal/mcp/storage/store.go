package storage

import (
	"context"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
)

// Store defines the interface for MCP configuration storage
type Store interface {
	// Create creates a new MCP configuration
	Create(ctx context.Context, cfg *config.MCPConfig) error

	// Get gets an MCP configuration by tenant and name
	Get(ctx context.Context, tenant, name string) (*config.MCPConfig, error)

	// List lists all MCP configurations
	// includeDeleted: if true, includes soft deleted records
	List(ctx context.Context, includeDeleted ...bool) ([]*config.MCPConfig, error)

	// ListUpdated lists all MCP configurations updated since a given time
	ListUpdated(ctx context.Context, since time.Time) ([]*config.MCPConfig, error)

	// Update updates an existing MCP configuration
	Update(ctx context.Context, cfg *config.MCPConfig) error

	// Delete deletes an MCP configuration by tenant and name
	Delete(ctx context.Context, tenant, name string) error

	// GetVersion gets a specific version of the configuration by tenant and name
	GetVersion(ctx context.Context, tenant, name string, version int) (*config.MCPConfigVersion, error)

	// ListVersions lists all versions of a configuration by tenant and name
	ListVersions(ctx context.Context, tenant, name string) ([]*config.MCPConfigVersion, error)

	// DeleteVersion deletes a specific version by tenant and name
	DeleteVersion(ctx context.Context, tenant, name string, version int) error

	// SetActiveVersion sets a specific version as the active version by tenant and name
	SetActiveVersion(ctx context.Context, tenant, name string, version int) error
}

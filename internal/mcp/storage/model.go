package storage

import (
	"encoding/json"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"gorm.io/gorm"
)

// MCPConfig represents the database model for MCPConfig
type MCPConfig struct {
	Name       string `gorm:"primaryKey; column:name"`
	Tenant     string `gorm:"column:tenant; default:''"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Routers    string `gorm:"type:text; column:routers"`
	Servers    string `gorm:"type:text; column:servers"`
	Tools      string `gorm:"type:text; column:tools"`
	McpServers string `gorm:"type:text; column:mcp_servers"`
}

// ToMCPConfig converts the database model to MCPConfig
func (m *MCPConfig) ToMCPConfig() (*config.MCPConfig, error) {
	cfg := &config.MCPConfig{
		Name:      m.Name,
		Tenant:    m.Tenant,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if len(m.Routers) > 0 {
		if err := json.Unmarshal([]byte(m.Routers), &cfg.Routers); err != nil {
			return nil, err
		}
	}
	if len(m.Servers) > 0 {
		if err := json.Unmarshal([]byte(m.Servers), &cfg.Servers); err != nil {
			return nil, err
		}
	}
	if len(m.Tools) > 0 {
		if err := json.Unmarshal([]byte(m.Tools), &cfg.Tools); err != nil {
			return nil, err
		}
	}
	if len(m.McpServers) > 0 {
		if err := json.Unmarshal([]byte(m.McpServers), &cfg.McpServers); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// FromMCPConfig converts MCPConfig to database model
func FromMCPConfig(cfg *config.MCPConfig) (*MCPConfig, error) {
	routers, err := json.Marshal(cfg.Routers)
	if err != nil {
		return nil, err
	}

	servers, err := json.Marshal(cfg.Servers)
	if err != nil {
		return nil, err
	}

	tools, err := json.Marshal(cfg.Tools)
	if err != nil {
		return nil, err
	}

	mcpServers, err := json.Marshal(cfg.McpServers)
	if err != nil {
		return nil, err
	}

	return &MCPConfig{
		Name:       cfg.Name,
		Tenant:     cfg.Tenant,
		CreatedAt:  cfg.CreatedAt,
		UpdatedAt:  cfg.UpdatedAt,
		Routers:    string(routers),
		Servers:    string(servers),
		Tools:      string(tools),
		McpServers: string(mcpServers),
	}, nil
}

// BeforeCreate is a GORM hook that sets timestamps
func (m *MCPConfig) BeforeCreate(_ *gorm.DB) error {
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate is a GORM hook that updates the UpdatedAt timestamp
func (m *MCPConfig) BeforeUpdate(_ *gorm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}

// ActiveVersion represents the currently active version of an MCP configuration
type ActiveVersion struct {
	ID        uint      `gorm:"primarykey"`
	Name      string    `gorm:"uniqueIndex;not null"`
	Version   int       `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// MCPConfigVersion represents the database model for MCPConfigVersion
type MCPConfigVersion struct {
	ID         int64           `gorm:"primaryKey;autoIncrement"`
	Name       string          `gorm:"column:name;index:idx_name_tenant_version,uniqueIndex"`
	Tenant     string          `gorm:"column:tenant;default:'';index:idx_name_tenant_version,uniqueIndex"`
	Version    int             `gorm:"column:version;index:idx_name_tenant_version,uniqueIndex"`
	ActionType cnst.ActionType `gorm:"column:action_type;not null"` // Create, Update, Delete, Revert
	CreatedBy  string          `gorm:"column:created_by"`
	CreatedAt  time.Time       `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)"`
	Routers    string          `gorm:"type:text;column:routers"`
	Servers    string          `gorm:"type:text;column:servers"`
	Tools      string          `gorm:"type:text;column:tools"`
	McpServers string          `gorm:"type:text;column:mcp_servers"`
}

// ToMCPConfig converts the database model to MCPConfig
func (m *MCPConfigVersion) ToMCPConfig() (*config.MCPConfig, error) {
	cfg := &config.MCPConfig{
		Name:      m.Name,
		Tenant:    m.Tenant,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.CreatedAt, // Use CreatedAt as UpdatedAt for versioned configs
	}

	if len(m.Routers) > 0 {
		if err := json.Unmarshal([]byte(m.Routers), &cfg.Routers); err != nil {
			return nil, err
		}
	}
	if len(m.Servers) > 0 {
		if err := json.Unmarshal([]byte(m.Servers), &cfg.Servers); err != nil {
			return nil, err
		}
	}
	if len(m.Tools) > 0 {
		if err := json.Unmarshal([]byte(m.Tools), &cfg.Tools); err != nil {
			return nil, err
		}
	}
	if len(m.McpServers) > 0 {
		if err := json.Unmarshal([]byte(m.McpServers), &cfg.McpServers); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// FromMCPConfigVersion converts MCPConfig to database model
func FromMCPConfigVersion(cfg *config.MCPConfig, version int, createdBy string, actionType cnst.ActionType) (*MCPConfigVersion, error) {
	routers, err := json.Marshal(cfg.Routers)
	if err != nil {
		return nil, err
	}

	servers, err := json.Marshal(cfg.Servers)
	if err != nil {
		return nil, err
	}

	tools, err := json.Marshal(cfg.Tools)
	if err != nil {
		return nil, err
	}

	mcpServers, err := json.Marshal(cfg.McpServers)
	if err != nil {
		return nil, err
	}

	return &MCPConfigVersion{
		Name:       cfg.Name,
		Tenant:     cfg.Tenant,
		Version:    version,
		ActionType: actionType,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
		Routers:    string(routers),
		Servers:    string(servers),
		Tools:      string(tools),
		McpServers: string(mcpServers),
	}, nil
}

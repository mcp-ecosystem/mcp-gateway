package storage

import (
	"encoding/json"
	"time"

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

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
	Name        string `gorm:"primaryKey; column:name"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ProtoType   string `gorm:"type:varchar(64); column:proto_type; required"`
	Routers     string `gorm:"type:text; column:routers; default:''"`
	Servers     string `gorm:"type:text; column:servers; default:''"`
	Tools       string `gorm:"type:text; column:tools; default:''"`
	StdioServer string `gorm:"type:text; column:stdio_server; default:''"`
	SSEServer   string `gorm:"type:text; column:sse_server; default:''"`
}

// ToMCPConfig converts the database model to MCPConfig
func (m *MCPConfig) ToMCPConfig() (*config.MCPConfig, error) {
	cfg := &config.MCPConfig{
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		ProtoType: cnst.ProtoType(m.ProtoType),
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
	if len(m.StdioServer) > 0 {
		if err := json.Unmarshal([]byte(m.StdioServer), &cfg.StdioServer); err != nil {
			return nil, err
		}
	}
	if len(m.SSEServer) > 0 {
		if err := json.Unmarshal([]byte(m.SSEServer), &cfg.SSEServer); err != nil {
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

	stdioConfigs, err := json.Marshal(cfg.StdioServer)
	if err != nil {
		return nil, err
	}

	sseConfigs, err := json.Marshal(cfg.SSEServer)
	if err != nil {
		return nil, err
	}

	return &MCPConfig{
		Name:        cfg.Name,
		CreatedAt:   cfg.CreatedAt,
		UpdatedAt:   cfg.UpdatedAt,
		ProtoType:   string(cfg.ProtoType),
		Routers:     string(routers),
		Servers:     string(servers),
		Tools:       string(tools),
		StdioServer: string(stdioConfigs),
		SSEServer:   string(sseConfigs),
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

package database

import (
	"time"
)

type RegistryServer struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(255)"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null;index"`
	Description string    `json:"description" gorm:"type:text"`
	TenantName  string    `json:"tenantName" gorm:"type:varchar(50);index"`
	Repository  string    `json:"repository" gorm:"type:text"` // JSON stored as text
	Version     string    `json:"version" gorm:"type:varchar(100)"`
	IsPublished bool      `json:"isPublished" gorm:"default:false"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type RegistryServerDetail struct {
	RegistryServer
	Packages  string `json:"packages" gorm:"type:text"`  // JSON stored as text
	Remotes   string `json:"remotes" gorm:"type:text"`   // JSON stored as text
	McpConfig string `json:"mcpConfig" gorm:"type:text"` // Full MCP configuration
}

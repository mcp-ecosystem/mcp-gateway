package storage

import (
	"context"
	"github.com/glebarez/sqlite"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"time"
)

// DBStore implements the Store interface using a database
type DBStore struct {
	logger *zap.Logger
	db     *gorm.DB
}

var _ Store = (*DBStore)(nil)

// DatabaseType represents the supported database types
type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgres"
	MySQL      DatabaseType = "mysql"
	SQLite     DatabaseType = "sqlite"
)

// NewDBStore creates a new database-based store
func NewDBStore(logger *zap.Logger, dbType DatabaseType, dsn string) (*DBStore, error) {
	logger = logger.Named("mcp.store.db")

	var dialector gorm.Dialector
	switch dbType {
	case PostgreSQL:
		dialector = postgres.Open(dsn)
	case MySQL:
		dialector = mysql.Open(dsn)
	case SQLite:
		dialector = sqlite.Open(dsn)
	default:
		return nil, ErrInvalidDatabaseType
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto migrate the schema
	if err := db.AutoMigrate(&MCPConfig{}); err != nil {
		return nil, err
	}

	return &DBStore{
		logger: logger,
		db:     db,
	}, nil
}

// Create implements Store.Create
func (s *DBStore) Create(_ context.Context, server *config.MCPConfig) error {
	model, err := FromMCPConfig(server)
	if err != nil {
		return err
	}

	result := s.db.Create(model)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Get implements Store.Get
func (s *DBStore) Get(_ context.Context, name string) (*config.MCPConfig, error) {
	var model MCPConfig
	result := s.db.Where("name = ?", name).First(&model)
	if result.Error != nil {
		return nil, result.Error
	}
	return model.ToMCPConfig()
}

// List implements Store.List
func (s *DBStore) List(_ context.Context) ([]*config.MCPConfig, error) {
	var models []MCPConfig
	result := s.db.Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	configs := make([]*config.MCPConfig, len(models))
	for i, model := range models {
		cfg, err := model.ToMCPConfig()
		if err != nil {
			return nil, err
		}
		configs[i] = cfg
	}
	return configs, nil
}

// Update implements Store.Update
func (s *DBStore) Update(_ context.Context, server *config.MCPConfig) error {
	model, err := FromMCPConfig(server)
	if err != nil {
		return err
	}

	result := s.db.Save(model)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Delete implements Store.Delete
func (s *DBStore) Delete(_ context.Context, name string) error {
	result := s.db.Where("name = ?", name).Delete(&MCPConfig{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *DBStore) GetValidMcpKey(_ context.Context, provider string) (string, error) {
	var model config.McpKey
	result := s.db.Where("provider = ?", provider).Where("is_valid = ?", true).First(&model)
	if result.Error != nil {
		return "", result.Error
	}
	return model.Key, nil
}

func (s *DBStore) InvalidateMcpKey(_ context.Context, provider, key string) error {
	result := s.db.Model(&config.McpKey{}).Where("provider = ?", provider).
		Where("key = ?", key).
		Updates(map[string]interface{}{
			"is_valid":   false,
			"updated_at": time.Now(),
		})
	return result.Error
}

// ErrInvalidDatabaseType is returned when an invalid database type is provided
var ErrInvalidDatabaseType = gorm.ErrInvalidDB

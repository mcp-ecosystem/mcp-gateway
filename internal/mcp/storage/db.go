package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		return nil, gorm.ErrInvalidDB
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto migrate the schema
	if err := db.AutoMigrate(&MCPConfig{}, &MCPConfigVersion{}, &ActiveVersion{}); err != nil {
		return nil, err
	}

	return &DBStore{
		logger: logger,
		db:     db,
	}, nil
}

// Create implements Store.Create
func (s *DBStore) Create(_ context.Context, server *config.MCPConfig) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		model, err := FromMCPConfig(server)
		if err != nil {
			return err
		}

		// Create the main record
		if err := tx.Create(model).Error; err != nil {
			return err
		}

		// Create version record
		version, err := FromMCPConfigVersion(server, 1, "system", cnst.ActionCreate)
		if err != nil {
			return err
		}

		if err := tx.Create(version).Error; err != nil {
			return err
		}

		// Create active version record
		activeVersion := &ActiveVersion{
			Name:      server.Name,
			Version:   1,
			UpdatedAt: time.Now(),
		}

		if err := tx.Create(activeVersion).Error; err != nil {
			return err
		}

		return nil
	})
}

// Get implements Store.Get
func (s *DBStore) Get(_ context.Context, name string) (*config.MCPConfig, error) {
	var model MCPConfig
	err := s.db.Where("name = ?", name).First(&model).Error
	if err != nil {
		return nil, err
	}
	return model.ToMCPConfig()
}

// List implements Store.List
func (s *DBStore) List(_ context.Context) ([]*config.MCPConfig, error) {
	var models []MCPConfig
	err := s.db.Find(&models).Error
	if err != nil {
		return nil, err
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
func (s *DBStore) Update(ctx context.Context, server *config.MCPConfig) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		model, err := FromMCPConfig(server)
		if err != nil {
			return err
		}

		// Update the main record
		if err := tx.Save(model).Error; err != nil {
			return err
		}

		// Get the latest version
		var latestVersion MCPConfigVersion
		if err := tx.Where("name = ?", server.Name).Order("version DESC").First(&latestVersion).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// Calculate hash for current config
		version, err := FromMCPConfigVersion(server, latestVersion.Version+1, "system", cnst.ActionUpdate)
		if err != nil {
			return err
		}

		// If there's a latest version and its hash matches the current config's hash, skip creating new version
		if latestVersion.Version > 0 && latestVersion.Hash == version.Hash {
			s.logger.Info("Skipping version creation as content hash matches latest version",
				zap.String("name", server.Name),
				zap.Int("version", latestVersion.Version),
				zap.String("hash", version.Hash))
			return nil
		}

		// Create new version
		if err := tx.Create(version).Error; err != nil {
			return err
		}

		// Update active version
		activeVersion := &ActiveVersion{
			Name:      server.Name,
			Version:   version.Version,
			UpdatedAt: time.Now(),
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"version", "updated_at"}),
		}).Create(activeVersion).Error; err != nil {
			return err
		}

		return nil
	})
}

// Delete implements Store.Delete
func (s *DBStore) Delete(ctx context.Context, name string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Get the config before deletion to create version record
		var config MCPConfig
		if err := tx.Where("name = ?", name).First(&config).Error; err != nil {
			return err
		}

		// Get the latest version number
		var latestVersion int
		if err := tx.Model(&MCPConfigVersion{}).
			Where("name = ?", name).
			Select("COALESCE(MAX(version), 0)").
			Scan(&latestVersion).Error; err != nil {
			return err
		}

		// Create version record for deletion
		mcpConfig, err := config.ToMCPConfig()
		if err != nil {
			return err
		}
		version, err := FromMCPConfigVersion(mcpConfig, latestVersion+1, "system", cnst.ActionDelete)
		if err != nil {
			return err
		}

		if err := tx.Create(version).Error; err != nil {
			return err
		}

		// Delete the active version
		if err := tx.Where("name = ?", name).Delete(&ActiveVersion{}).Error; err != nil {
			return err
		}

		// Delete the main record
		if err := tx.Where("name = ?", name).Delete(&MCPConfig{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetVersion gets a specific version of the configuration
func (s *DBStore) GetVersion(ctx context.Context, name string, version int) (*config.MCPConfigVersion, error) {
	var versionModel MCPConfigVersion
	if err := s.db.Where("name = ? AND version = ?", name, version).First(&versionModel).Error; err != nil {
		return nil, err
	}
	return &config.MCPConfigVersion{
		Version:    versionModel.Version,
		CreatedBy:  versionModel.CreatedBy,
		CreatedAt:  versionModel.CreatedAt,
		ActionType: versionModel.ActionType,
		Name:       versionModel.Name,
		Tenant:     versionModel.Tenant,
		Routers:    versionModel.Routers,
		Servers:    versionModel.Servers,
		Tools:      versionModel.Tools,
		McpServers: versionModel.McpServers,
		Hash:       versionModel.Hash,
	}, nil
}

// ListVersions lists all versions of a configuration
func (s *DBStore) ListVersions(ctx context.Context, name string) ([]*config.MCPConfigVersion, error) {
	var versions []MCPConfigVersion
	if err := s.db.Where("name = ?", name).Order("version DESC").Find(&versions).Error; err != nil {
		return nil, err
	}

	// Get active version
	var activeVersion ActiveVersion
	if err := s.db.Where("name = ?", name).First(&activeVersion).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	result := make([]*config.MCPConfigVersion, len(versions))
	for i, v := range versions {
		result[i] = &config.MCPConfigVersion{
			Version:    v.Version,
			CreatedBy:  v.CreatedBy,
			CreatedAt:  v.CreatedAt,
			ActionType: v.ActionType,
			Name:       v.Name,
			Tenant:     v.Tenant,
			Routers:    v.Routers,
			Servers:    v.Servers,
			Tools:      v.Tools,
			McpServers: v.McpServers,
			IsActive:   v.Version == activeVersion.Version,
			Hash:       v.Hash,
		}
	}
	return result, nil
}

// DeleteVersion deletes a specific version
func (s *DBStore) DeleteVersion(ctx context.Context, name string, version int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Check if this is the active version
		var activeVersion ActiveVersion
		if err := tx.Where("name = ? AND version = ?", name, version).First(&activeVersion).Error; err == nil {
			return fmt.Errorf("cannot delete active version")
		}

		// Delete the version
		if err := tx.Where("name = ? AND version = ?", name, version).Delete(&MCPConfigVersion{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// SetActiveVersion sets a specific version as the active version
func (s *DBStore) SetActiveVersion(ctx context.Context, name string, version int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Check if the version exists
		var versionModel MCPConfigVersion
		if err := tx.Where("name = ? AND version = ?", name, version).First(&versionModel).Error; err != nil {
			return fmt.Errorf("version %d not found: %w", version, err)
		}

		// Get the latest version number
		var latestVersion int
		if err := tx.Model(&MCPConfigVersion{}).
			Where("name = ?", name).
			Select("COALESCE(MAX(version), 0)").
			Scan(&latestVersion).Error; err != nil {
			return err
		}

		// Create new version record for revert action
		newVersion := &MCPConfigVersion{
			Name:       versionModel.Name,
			Tenant:     versionModel.Tenant,
			Version:    latestVersion + 1,
			ActionType: cnst.ActionRevert,
			CreatedBy:  "system",
			CreatedAt:  time.Now(),
			Routers:    versionModel.Routers,
			Servers:    versionModel.Servers,
			Tools:      versionModel.Tools,
			McpServers: versionModel.McpServers,
		}

		if err := tx.Create(newVersion).Error; err != nil {
			return err
		}

		// Update or create active version
		activeVersion := &ActiveVersion{
			Name:      name,
			Version:   newVersion.Version,
			UpdatedAt: time.Now(),
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"version", "updated_at"}),
		}).Create(activeVersion).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *DBStore) ListRegistryServers(ctx context.Context, cursor string, limit int, tenantName string) ([]*config.MCPConfig, string, error) {
	var models []MCPConfig
	query := s.db.Where("is_published = ?", true)
	
	if tenantName != "" {
		query = query.Where("tenant = ?", tenantName)
	}
	
	if cursor != "" {
		var cursorModel MCPConfig
		if err := s.db.Where("id = ?", cursor).First(&cursorModel).Error; err != nil {
			return nil, "", err
		}
		query = query.Where("created_at > ?", cursorModel.CreatedAt)
	}
	
	if limit > 0 {
		query = query.Limit(limit + 1) // +1 to check if there are more results
	}
	
	if err := query.Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, "", err
	}
	
	var nextCursor string
	if limit > 0 && len(models) > limit {
		nextCursor = models[limit].ID
		models = models[:limit]
	}
	
	configs := make([]*config.MCPConfig, len(models))
	for i, model := range models {
		cfg, err := model.ToMCPConfig()
		if err != nil {
			return nil, "", err
		}
		configs[i] = cfg
	}
	
	return configs, nextCursor, nil
}

func (s *DBStore) GetRegistryServerByID(ctx context.Context, id string, tenantName string) (*config.MCPConfig, error) {
	var model MCPConfig
	query := s.db.Where("id = ? AND is_published = ?", id, true)
	
	if tenantName != "" {
		query = query.Where("tenant = ?", tenantName)
	}
	
	if err := query.First(&model).Error; err != nil {
		return nil, err
	}
	
	return model.ToMCPConfig()
}

func (s *DBStore) PublishToRegistry(ctx context.Context, cfg *config.MCPConfig) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		model, err := FromMCPConfig(cfg)
		if err != nil {
			return err
		}
		
		if model.ID == "" {
			model.ID = uuid.New().String()
		}
		
		model.IsPublished = true
		
		if err := tx.Save(model).Error; err != nil {
			return err
		}
		
		// Update the cfg with the generated ID
		cfg.ID = model.ID
		
		return nil
	})
}

func (s *DBStore) UnpublishFromRegistry(ctx context.Context, name string, tenantName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var model MCPConfig
		if err := tx.Where("name = ?", name).First(&model).Error; err != nil {
			return err
		}
		
		if tenantName != "" && model.Tenant != tenantName {
			return fmt.Errorf("unauthorized: server belongs to a different tenant")
		}
		
		model.IsPublished = false
		
		if err := tx.Save(&model).Error; err != nil {
			return err
		}
		
		return nil
	})
}

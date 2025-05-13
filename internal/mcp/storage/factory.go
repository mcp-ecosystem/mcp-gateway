package storage

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
)

// NewStore creates a new store based on configuration
func NewStore(logger *zap.Logger, cfg *config.StorageConfig) (Store, error) {
	logger.Info("Initializing storage", zap.String("type", cfg.Type))
	switch cfg.Type {
	case "disk":
		return NewDiskStore(logger, cfg.Disk.Path)
	case "db":
		dsn, err := buildDSN(&cfg.Database)
		if err != nil {
			return nil, err
		}
		return NewDBStore(logger, DatabaseType(cfg.Database.Type), dsn)
	case "api":
		return NewAPIStore(logger, cfg.API.Url, cfg.API.ConfigJSONPath, cfg.API.Timeout)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}

// buildDSN builds the database connection string based on configuration
func buildDSN(cfg *config.DatabaseConfig) (string, error) {
	switch cfg.Type {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host,
			cfg.Port,
			cfg.User,
			cfg.Password,
			cfg.DBName,
			cfg.SSLMode), nil
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.DBName), nil
	case "sqlite":
		return cfg.DBName, nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

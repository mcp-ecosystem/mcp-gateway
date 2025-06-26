package storage

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/amoylab/unla/internal/common/config"
)

// NewStore creates a new auth store based on configuration
func NewStore(logger *zap.Logger, cfg *config.OAuth2StorageConfig) (Store, error) {
	logger.Info("Initializing auth storage", zap.String("type", cfg.Type))
	switch cfg.Type {
	case "memory":
		return NewMemoryStorage(), nil
	case "redis":
		return NewRedisStorage(cfg.Redis.ClusterType, cfg.Redis.Addr, cfg.Redis.MasterName, cfg.Redis.Username, cfg.Redis.Password, cfg.Redis.DB)
	default:
		return nil, fmt.Errorf("unsupported auth storage type: %s", cfg.Type)
	}
}

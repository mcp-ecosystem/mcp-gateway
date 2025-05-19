package session

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
)

// Type represents the type of session store
type Type string

const (
	// TypeMemory represents in-memory session store
	TypeMemory Type = "memory"
	// TypeRedis represents Redis-based session store
	TypeRedis Type = "redis"
)

// NewStore creates a new session store based on configuration
func NewStore(logger *zap.Logger, cfg *config.SessionConfig) (Store, error) {
	logger.Info("Initializing session store", zap.String("type", cfg.Type))
	switch Type(cfg.Type) {
	case TypeMemory:
		return NewMemoryStore(logger), nil
	case TypeRedis:
		return NewRedisStore(logger, cfg.Redis)
	default:
		return nil, fmt.Errorf("unsupported session store type: %s", cfg.Type)
	}
}

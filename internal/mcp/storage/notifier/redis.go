package notifier

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
)

// RedisNotifier implements Notifier using Redis pub/sub
type RedisNotifier struct {
	logger *zap.Logger
	client *redis.Client
	topic  string
	role   config.NotifierRole
}

// NewRedisNotifier creates a new Redis-based notifier
func NewRedisNotifier(logger *zap.Logger, addr, username, password string, db int, topic string, role config.NotifierRole) (*RedisNotifier, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
	})

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisNotifier{
		logger: logger.Named("notifier.redis"),
		client: client,
		topic:  topic,
		role:   role,
	}, nil
}

// Watch implements Notifier.Watch
func (r *RedisNotifier) Watch(ctx context.Context) (<-chan *config.MCPConfig, error) {
	if !r.CanReceive() {
		return nil, cnst.ErrNotReceiver
	}

	ch := make(chan *config.MCPConfig, 10)

	pubsub := r.client.Subscribe(ctx, r.topic)
	go func() {
		defer close(ch)
		defer pubsub.Close()

		for msg := range pubsub.Channel() {
			var cfg config.MCPConfig
			if err := json.Unmarshal([]byte(msg.Payload), &cfg); err == nil {
				select {
				case ch <- &cfg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// NotifyUpdate implements Notifier.NotifyUpdate
func (r *RedisNotifier) NotifyUpdate(ctx context.Context, server *config.MCPConfig) error {
	if !r.CanSend() {
		return cnst.ErrNotSender
	}

	data, err := json.Marshal(server)
	if err != nil {
		return fmt.Errorf("failed to marshal server config: %w", err)
	}

	return r.client.Publish(ctx, r.topic, data).Err()
}

// CanReceive returns true if the notifier can receive updates
func (r *RedisNotifier) CanReceive() bool {
	return r.role == config.RoleReceiver || r.role == config.RoleBoth
}

// CanSend returns true if the notifier can send updates
func (r *RedisNotifier) CanSend() bool {
	return r.role == config.RoleSender || r.role == config.RoleBoth
}

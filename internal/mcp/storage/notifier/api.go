package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/amoylab/unla/internal/common/cnst"

	"github.com/amoylab/unla/internal/common/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// APINotifier implements Notifier using HTTP API
type APINotifier struct {
	logger    *zap.Logger
	watchers  map[chan<- *config.MCPConfig]struct{}
	mu        sync.RWMutex
	router    *gin.Engine
	server    *http.Server
	role      config.NotifierRole
	targetURL string
}

// NewAPINotifier creates a new API-based notifier
func NewAPINotifier(logger *zap.Logger, port int, role config.NotifierRole, targetURL string) *APINotifier {
	n := &APINotifier{
		logger:    logger.Named("notifier.api"),
		watchers:  make(map[chan<- *config.MCPConfig]struct{}),
		router:    gin.Default(),
		role:      role,
		targetURL: targetURL,
	}

	// Setup API routes if can receive
	if n.CanReceive() {
		n.router.POST("/_reload", func(c *gin.Context) {
			var mcpConfig *config.MCPConfig

			if c.Request.ContentLength > 0 {
				var cfg config.MCPConfig
				if err := c.ShouldBindJSON(&cfg); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": fmt.Sprintf("Invalid request body: %v", err)})
					return
				}
				mcpConfig = &cfg
				n.logger.Info("Received configuration update with reload", zap.String("config_name", cfg.Name))
			} else {
				n.logger.Info("Received reload signal without configuration")
			}

			if mcpConfig != nil {
				n.mu.RLock()
				for ch := range n.watchers {
					select {
					case ch <- mcpConfig:
						// Successfully sent
					default:
						n.logger.Warn("Watcher channel is full, skipping notification")
					}
				}
				n.mu.RUnlock()
			} else {
				n.mu.RLock()
				for ch := range n.watchers {
					select {
					case ch <- nil:
						// Successfully sent
					default:
						n.logger.Warn("Watcher channel is full, skipping notification")
					}
				}
				n.mu.RUnlock()
			}

			c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Reload triggered"})
		})

		// Start HTTP server
		n.server = &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: n.router,
		}

		go func() {
			if err := n.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				n.logger.Fatal("failed to start API server", zap.Error(err))
			}
		}()
	}

	return n
}

// Watch implements Notifier.Watch
func (n *APINotifier) Watch(ctx context.Context) (<-chan *config.MCPConfig, error) {
	if !n.CanReceive() {
		return nil, cnst.ErrNotReceiver
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	ch := make(chan *config.MCPConfig, 10)
	n.watchers[ch] = struct{}{}

	// Start a goroutine to handle context cancellation and cleanup
	go func() {
		<-ctx.Done()
		n.mu.Lock()
		defer n.mu.Unlock()
		delete(n.watchers, ch)
		close(ch)
	}()

	return ch, nil
}

// NotifyUpdate implements Notifier.NotifyUpdate
func (n *APINotifier) NotifyUpdate(ctx context.Context, server *config.MCPConfig) error {
	if !n.CanSend() {
		return cnst.ErrNotSender
	}

	if n.targetURL == "" {
		return fmt.Errorf("target URL is not configured")
	}

	var req *http.Request
	var err error

	reloadURL := n.targetURL
	if !strings.HasSuffix(reloadURL, "/_reload") {
		if !strings.HasSuffix(reloadURL, "/") {
			reloadURL += "/"
		}
		reloadURL += "_reload"
	}

	if server == nil {
		req, err = http.NewRequestWithContext(ctx, "POST", reloadURL, nil)
	} else {
		var body []byte
		body, err = json.Marshal(server)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		req, err = http.NewRequestWithContext(ctx, "POST", reloadURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	}

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unexpected status code: %d, failed to read body: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Shutdown gracefully shuts down the API server
func (n *APINotifier) Shutdown(ctx context.Context) error {
	if n.server != nil {
		return n.server.Shutdown(ctx)
	}
	return nil
}

// CanReceive returns true if the notifier can receive updates
func (n *APINotifier) CanReceive() bool {
	return n.role == config.RoleReceiver || n.role == config.RoleBoth
}

// CanSend returns true if the notifier can send updates
func (n *APINotifier) CanSend() bool {
	return n.role == config.RoleSender || n.role == config.RoleBoth
}

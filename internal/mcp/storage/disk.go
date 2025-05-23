package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"gopkg.in/yaml.v3"

	"go.uber.org/zap"
)

type DiskStore struct {
	logger  *zap.Logger
	baseDir string
	mu      sync.RWMutex
}

var _ Store = (*DiskStore)(nil)

// NewDiskStore creates a new disk-based store
func NewDiskStore(logger *zap.Logger, baseDir string) (*DiskStore, error) {
	logger = logger.Named("mcp.store")

	baseDir = getConfigPath(baseDir)
	logger.Info("Using configuration directory", zap.String("path", baseDir))

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	return &DiskStore{
		logger:  logger,
		baseDir: baseDir,
	}, nil
}

func (s *DiskStore) Create(_ context.Context, server *config.MCPConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	server.UpdatedAt = time.Now()
	if server.CreatedAt.IsZero() {
		server.CreatedAt = server.UpdatedAt
	}

	data, err := yaml.Marshal(server)
	if err != nil {
		return err
	}

	path := filepath.Join(s.baseDir, server.Name+".yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (s *DiskStore) Get(_ context.Context, name string) (*config.MCPConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.baseDir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var server config.MCPConfig
	if err := yaml.Unmarshal(data, &server); err != nil {
		return nil, err
	}

	return &server, nil
}

func (s *DiskStore) List(_ context.Context) ([]*config.MCPConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var servers []*config.MCPConfig
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.baseDir, entry.Name()))
		if err != nil {
			s.logger.Error("failed to read server file",
				zap.String("file", entry.Name()),
				zap.Error(err))
			continue
		}

		var server config.MCPConfig
		if err := yaml.Unmarshal(data, &server); err != nil {
			s.logger.Error("failed to unmarshal server",
				zap.String("file", entry.Name()),
				zap.Error(err))
			continue
		}

		servers = append(servers, &server)
	}

	return servers, nil
}

func (s *DiskStore) Update(_ context.Context, server *config.MCPConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	server.UpdatedAt = time.Now()
	data, err := yaml.Marshal(server)
	if err != nil {
		return err
	}

	path := filepath.Join(s.baseDir, server.Name+".yaml")
	return os.WriteFile(path, data, 0644)
}

func (s *DiskStore) Delete(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.baseDir, name+".yaml")
	return os.Remove(path)
}

func (s *DiskStore) GetVersion(_ context.Context, name string, version int) (*config.MCPConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versionPath := filepath.Join(s.baseDir, "versions", name, fmt.Sprintf("%d.yaml", version))
	data, err := os.ReadFile(versionPath)
	if err != nil {
		return nil, err
	}

	var model MCPConfigVersion
	if err := yaml.Unmarshal(data, &model); err != nil {
		return nil, err
	}

	return &config.MCPConfigVersion{
		Version:    model.Version,
		CreatedBy:  model.CreatedBy,
		CreatedAt:  model.CreatedAt,
		ActionType: model.ActionType,
		Name:       model.Name,
		Tenant:     model.Tenant,
		Routers:    model.Routers,
		Servers:    model.Servers,
		Tools:      model.Tools,
		McpServers: model.McpServers,
	}, nil
}

func (s *DiskStore) ListVersions(_ context.Context, name string) ([]*config.MCPConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versionsDir := filepath.Join(s.baseDir, "versions", name)
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var versions []*config.MCPConfigVersion
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(versionsDir, entry.Name()))
		if err != nil {
			s.logger.Error("failed to read version file",
				zap.String("file", entry.Name()),
				zap.Error(err))
			continue
		}

		var model MCPConfigVersion
		if err := yaml.Unmarshal(data, &model); err != nil {
			s.logger.Error("failed to unmarshal version",
				zap.String("file", entry.Name()),
				zap.Error(err))
			continue
		}

		versions = append(versions, &config.MCPConfigVersion{
			Version:    model.Version,
			CreatedBy:  model.CreatedBy,
			CreatedAt:  model.CreatedAt,
			ActionType: model.ActionType,
			Name:       model.Name,
			Tenant:     model.Tenant,
			Routers:    model.Routers,
			Servers:    model.Servers,
			Tools:      model.Tools,
			McpServers: model.McpServers,
		})
	}

	// Sort versions by version number in descending order
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})

	return versions, nil
}

func (s *DiskStore) DeleteVersion(_ context.Context, name string, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	versionPath := filepath.Join(s.baseDir, "versions", name, fmt.Sprintf("%d.yaml", version))
	return os.Remove(versionPath)
}

func (s *DiskStore) GetActiveVersion(_ context.Context, name string) (*config.MCPConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	versionsDir := filepath.Join(s.baseDir, "versions", name)
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return nil, err
	}

	// Sort entries by version number in descending order
	sort.Slice(entries, func(i, j int) bool {
		vi, _ := strconv.Atoi(strings.TrimSuffix(entries[i].Name(), ".yaml"))
		vj, _ := strconv.Atoi(strings.TrimSuffix(entries[j].Name(), ".yaml"))
		return vi > vj
	})

	// Get the latest version
	if len(entries) > 0 {
		data, err := os.ReadFile(filepath.Join(versionsDir, entries[0].Name()))
		if err != nil {
			return nil, err
		}

		var model MCPConfigVersion
		if err := yaml.Unmarshal(data, &model); err != nil {
			return nil, err
		}

		return model.ToMCPConfig()
	}

	return nil, fmt.Errorf("no version found for %s", name)
}

func (s *DiskStore) SetActiveVersion(_ context.Context, name string, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	versionsDir := filepath.Join(s.baseDir, "versions", name)
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return err
	}

	// First, deactivate all versions
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(versionsDir, entry.Name()))
		if err != nil {
			continue
		}

		var model MCPConfigVersion
		if err := yaml.Unmarshal(data, &model); err != nil {
			continue
		}

		// Create new version record for revert action
		cfg, err := model.ToMCPConfig()
		if err != nil {
			return err
		}
		newVersion, err := FromMCPConfigVersion(cfg, model.Version, "system", cnst.ActionRevert)
		if err != nil {
			return err
		}

		newData, err := yaml.Marshal(newVersion)
		if err != nil {
			return err
		}
		err = os.WriteFile(filepath.Join(versionsDir, entry.Name()), newData, 0644)
		if err != nil {
			return err
		}
	}

	// Then, activate the specified version
	versionPath := filepath.Join(versionsDir, fmt.Sprintf("%d.yaml", version))
	data, err := os.ReadFile(versionPath)
	if err != nil {
		return err
	}

	var model MCPConfigVersion
	if err := yaml.Unmarshal(data, &model); err != nil {
		return err
	}

	// Create new version record for revert action
	cfg, err := model.ToMCPConfig()
	if err != nil {
		return err
	}
	newVersion, err := FromMCPConfigVersion(cfg, model.Version, "system", cnst.ActionRevert)
	if err != nil {
		return err
	}

	newData, err := yaml.Marshal(newVersion)
	if err != nil {
		return err
	}

	return os.WriteFile(versionPath, newData, 0644)
}

func getConfigPath(baseDir string) string {
	// 1. Check command line flag
	if baseDir != "" {
		return baseDir
	}

	// 2. Default to APPDATA/.mcp/gateway
	appData := os.Getenv("APPDATA")
	if appData == "" {
		// For non-Windows systems, use HOME
		appData = os.Getenv("HOME")
		if appData == "" {
			log.Fatal("Neither APPDATA nor HOME environment variable is set")
		}
	}
	return filepath.Join(appData, ".mcp", "gateway")
}

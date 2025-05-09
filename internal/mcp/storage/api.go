package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/tidwall/gjson"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"

	"go.uber.org/zap"
)

// APIStore implements the Store interface using the remote http server
type APIStore struct {
	logger *zap.Logger
	url    string
	// read config from response(json body) using gjson
	configJSONPath string
	timeout        int
}

var _ Store = (*APIStore)(nil)

// NewAPIStore creates a new api-based store
func NewAPIStore(logger *zap.Logger, url string, configJSONPath string, timeout int) (*APIStore, error) {
	logger = logger.Named("mcp.store")

	logger.Info("Using configuration url", zap.String("path", url))

	return &APIStore{
		logger:         logger,
		url:            url,
		configJSONPath: configJSONPath,
		timeout:        timeout,
	}, nil
}

// Create implements Store.Create
func (s *APIStore) Create(_ context.Context, server *config.MCPConfig) error {
	// only use for read config
	return nil
}

// Get implements Store.Get
func (s *APIStore) Get(_ context.Context, name string) (*config.MCPConfig, error) {
	jsonStr, err := s.request()
	if err != nil {
		return nil, err
	}
	var config config.MCPConfig
	err = json.Unmarshal([]byte(jsonStr), &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// List implements Store.List
func (s *APIStore) List(_ context.Context) ([]*config.MCPConfig, error) {
	jsonStr, err := s.request()
	if err != nil {
		return nil, err
	}
	var configs []*config.MCPConfig
	err = json.Unmarshal([]byte(jsonStr), &configs)
	if err != nil {
		return nil, err
	}
	return configs, nil
}

// Update implements Store.Update
func (s *APIStore) Update(_ context.Context, server *config.MCPConfig) error {
	// only use for read config
	return nil
}

// Delete implements Store.Delete
func (s *APIStore) Delete(_ context.Context, name string) error {
	// only use for read config
	return nil
}

func (s *APIStore) request() (string, error) {
	client := &http.Client{
		Timeout: time.Duration(s.timeout) * time.Second,
	}
	resp, err := client.Get(s.url)
	if err != nil {
		s.logger.Error("failed to request url",
			zap.String("url", s.url),
			zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("failed to read response",
			zap.String("url", s.url),
			zap.Error(err))
		return "", err
	}
	jsonString := string(body)
	s.logger.Debug("read storage api response", zap.String("body", jsonString))
	if s.configJSONPath == "" {
		return jsonString, nil
	}
	result := gjson.Get(jsonString, s.configJSONPath)
	if !result.Exists() {
		err = fmt.Errorf("configJSONPath is not in response: %s", s.configJSONPath)
		s.logger.Error("configJSONPath is not in response",
			zap.String("url", s.url),
			zap.Error(err))
		return "", err
	}
	return result.Raw, nil
}

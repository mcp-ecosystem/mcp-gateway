package config

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/pkg/helper"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

type (
	// SuperAdminConfig represents the super admin configuration
	SuperAdminConfig struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	}

	// MCPGatewayConfig represents the MCP gateway configuration
	MCPGatewayConfig struct {
		Port           int              `yaml:"port"`
		ReloadPort     int              `yaml:"reload_port"`
		ReloadInterval time.Duration    `yaml:"reload_interval"`
		ReloadSwitch   bool             `yaml:"reload_switch"`
		PID            string           `yaml:"pid"`
		SuperAdmin     SuperAdminConfig `yaml:"super_admin"`
		Logger         LoggerConfig     `yaml:"logger"`
		Storage        StorageConfig    `yaml:"storage"`
		Notifier       NotifierConfig   `yaml:"notifier"`
		Session        SessionConfig    `yaml:"session"`
	}

	// SessionConfig represents the session storage configuration
	SessionConfig struct {
		Type  string             `yaml:"type"`  // "memory" or "redis"
		Redis SessionRedisConfig `yaml:"redis"` // Redis configuration
	}

	// SessionRedisConfig represents the Redis configuration for session storage
	SessionRedisConfig struct {
		Addr     string `yaml:"addr"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
		Topic    string `yaml:"topic"`
		Prefix   string `yaml:"prefix"`
	}

	// LoggerConfig represents the logger configuration
	LoggerConfig struct {
		Level      string `yaml:"level"`       // debug, info, warn, error
		Format     string `yaml:"format"`      // json, console
		Output     string `yaml:"output"`      // stdout, file
		FilePath   string `yaml:"file_path"`   // path to log file when output is file
		MaxSize    int    `yaml:"max_size"`    // max size of log file in MB
		MaxBackups int    `yaml:"max_backups"` // max number of backup files
		MaxAge     int    `yaml:"max_age"`     // max age of backup files in days
		Compress   bool   `yaml:"compress"`    // whether to compress backup files
		Color      bool   `yaml:"color"`       // whether to use color in console output
		Stacktrace bool   `yaml:"stacktrace"`  // whether to include stacktrace in error logs
		TimeZone   string `yaml:"time_zone"`   // time zone for log timestamps, e.g., "UTC", default is local
		TimeFormat string `yaml:"time_format"` // time format for log timestamps, default is "2006-01-02 15:04:05"
	}
)

type Type interface {
	MCPGatewayConfig | APIServerConfig
}

// LoadConfig loads configuration from a YAML file with environment variable support
func LoadConfig[T Type](filename string) (*T, string, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	cfgPath := helper.GetCfgPath(filename)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, cfgPath, err
	}

	// Resolve environment variables
	data = resolveEnv(data)

	var cfg T
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, cfgPath, err
	}

	// Validate durations after unmarshalling
	if mcpCfg, ok := any(cfg).(*MCPGatewayConfig); ok {
		if mcpCfg.ReloadInterval <= time.Second {
			mcpCfg.ReloadInterval = 600 * time.Second
		}
	}

	return &cfg, cfgPath, nil
}

// resolveEnv replaces environment variable placeholders in YAML content
func resolveEnv(content []byte) []byte {
	regex := regexp.MustCompile(`\$\{(\w+)(?::([^}]*))?\}`)

	return regex.ReplaceAllFunc(content, func(match []byte) []byte {
		matches := regex.FindSubmatch(match)
		envKey := string(matches[1])
		var defaultValue string

		if len(matches) > 2 {
			defaultValue = string(matches[2])
		}

		if value, exists := os.LookupEnv(envKey); exists {
			return []byte(value)
		}
		return []byte(defaultValue)
	})
}

// LoadAPIServerConfig loads the API server configuration from a file
func LoadAPIServerConfig(path string) (*APIServerConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg APIServerConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// NewLogger creates a new logger based on configuration
func NewLogger(cfg *LoggerConfig) (*zap.Logger, error) {
	var config zap.Config

	// Set log level
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// Configure encoder
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create logger configuration
	if cfg.Format == "json" {
		config = zap.Config{
			Level:             zap.NewAtomicLevelAt(level),
			Development:       false,
			DisableCaller:     false,
			DisableStacktrace: !cfg.Stacktrace,
			Sampling: &zap.SamplingConfig{
				Initial:    100,
				Thereafter: 100,
			},
			Encoding:         "json",
			EncoderConfig:    encoderConfig,
			OutputPaths:      []string{cfg.Output},
			ErrorOutputPaths: []string{cfg.Output},
		}
	} else {
		config = zap.Config{
			Level:             zap.NewAtomicLevelAt(level),
			Development:       false,
			DisableCaller:     false,
			DisableStacktrace: !cfg.Stacktrace,
			Sampling: &zap.SamplingConfig{
				Initial:    100,
				Thereafter: 100,
			},
			Encoding:         "console",
			EncoderConfig:    encoderConfig,
			OutputPaths:      []string{cfg.Output},
			ErrorOutputPaths: []string{cfg.Output},
		}
	}

	// Create logger
	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return logger, nil
}

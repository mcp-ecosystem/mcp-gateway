package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// the time zone sync.Once
	timezoneSyncOnce sync.Once
	//timezone time zone location
	timezone *time.Location
	// defaultZapOpts default zap options
	defaultZapOpts = []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	}
)

// NewLogger creates a new logger based on configuration
func NewLogger(cfg *config.LoggerConfig) (*zap.Logger, error) {
	// 设置默认配置
	setLoggerDefaults(cfg)
	// Create encoder config
	encoder := getEncoder(cfg)
	var syncer zapcore.WriteSyncer
	if cfg.Output == "file" {
		// Ensure log directory exists
		if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0755); err != nil {
			return nil, err
		}
		syncer = getLogWriter(cfg)
	} else {
		syncer = zap.CombineWriteSyncers(syncer, os.Stdout)
	}

	level := getLogLevel(cfg.Level)
	if level < zapcore.DebugLevel || level > zapcore.FatalLevel {
		level = zapcore.InfoLevel
	}

	logger := zap.New(
		zapcore.NewCore(
			encoder,
			syncer,
			level,
		),
		defaultZapOpts...,
	)

	// Add stacktrace if enabled
	if cfg.Stacktrace {
		logger = logger.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel))
	}

	return logger, nil
}

// setLoggerDefaults 设置日志配置的默认值
func setLoggerDefaults(cfg *config.LoggerConfig) {
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "json"
	}
	if cfg.Output == "" {
		cfg.Output = "stdout"
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 100 // 100MB
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 3
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 7 // 7 days
	}
	if cfg.TimeZone == "" {
		cfg.TimeZone = "Local"
	}
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = "2006-01-02 15:04:05"
	}
}

// getEncoder creates a zapcore.Encoder based on the configuration
func getEncoder(cfg *config.LoggerConfig) zapcore.Encoder {
	// Create encoder config
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
	// Set color if enabled
	if cfg.Color && cfg.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.In(resolveTimeZone(cfg)).Format(cfg.TimeFormat))
	}
	if cfg.Format == "json" {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// resolveTimeZone resolves the timezone based on the configuration
func resolveTimeZone(cfg *config.LoggerConfig) *time.Location {
	timezoneSyncOnce.Do(func() {
		if len(cfg.TimeZone) <= 0 {
			timezone = time.Local
			return
		}
		// Get timezone location
		var err error
		timezone, err = time.LoadLocation(cfg.TimeZone)
		if err != nil || timezone == nil {
			timezone = time.Local
		}
	})
	return timezone
}

// getLogWriter creates a lumberjack logger for file output
func getLogWriter(cfg *config.LoggerConfig) zapcore.WriteSyncer {
	hook := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		LocalTime:  true,
		Compress:   cfg.Compress,
	}
	return zapcore.AddSync(hook)
}

// getLogLevel converts string level to zapcore.Level
// debug:-1 info:0 warn:1 error:2 dpanic:3 panic:4 fatal:5
// default: INFO
func getLogLevel(level string) zapcore.Level {
	level = strings.ToLower(level)
	levelInt := zapcore.InfoLevel
	if level == "debug" {
		levelInt = zapcore.DebugLevel
	} else if level == "info" {
		levelInt = zapcore.InfoLevel
	} else if level == "warn" {
		levelInt = zapcore.WarnLevel
	} else if level == "error" {
		levelInt = zapcore.ErrorLevel
	} else if level == "dpanic" {
		levelInt = zapcore.DPanicLevel
	} else if level == "panic" {
		levelInt = zapcore.PanicLevel
	} else if level == "fatal" {
		levelInt = zapcore.FatalLevel
	}
	return levelInt
}

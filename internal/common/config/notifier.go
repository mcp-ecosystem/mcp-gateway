package config

type (
	// NotifierConfig represents the configuration for notifier
	NotifierConfig struct {
		Role   string       `yaml:"role"` // receiver, sender, or both
		Type   string       `yaml:"type"`
		Signal SignalConfig `yaml:"signal"`
		API    APIConfig    `yaml:"api"`
		Redis  RedisConfig  `yaml:"redis"`
	}

	// SignalConfig represents the configuration for signal-based notifier
	SignalConfig struct {
		Signal string `yaml:"signal"`
		PID    string `yaml:"pid"`
	}

	// APIConfig represents the configuration for API-based notifier
	APIConfig struct {
		Port      int    `yaml:"port"`
		TargetURL string `yaml:"target_url"`
	}

	// RedisConfig represents the configuration for Redis-based notifier
	RedisConfig struct {
		Addr     string `yaml:"addr"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
		Topic    string `yaml:"topic"`
	}
)

// NotifierRole represents the role of a notifier
type NotifierRole string

const (
	// RoleReceiver represents a notifier that can only receive updates
	RoleReceiver NotifierRole = "receiver"
	// RoleSender represents a notifier that can only send updates
	RoleSender NotifierRole = "sender"
	// RoleBoth represents a notifier that can both send and receive updates
	RoleBoth NotifierRole = "both"
)

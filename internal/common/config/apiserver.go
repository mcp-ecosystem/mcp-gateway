package config

import "fmt"

type (
	APIServerConfig struct {
		Database DatabaseConfig `yaml:"database"`
		OpenAI   OpenAIConfig   `yaml:"openai"`
		Storage  StorageConfig  `yaml:"storage"`
		Notifier NotifierConfig `yaml:"notifier"`
		Logger   LoggerConfig   `yaml:"logger"`
	}

	DatabaseConfig struct {
		Type     string `yaml:"type"`     // mysql, postgres, sqlite, etc.
		Host     string `yaml:"host"`     // localhost
		Port     int    `yaml:"port"`     // 3306 (for mysql), 5432 (for postgres)
		User     string `yaml:"user"`     // root (for mysql), postgres (for postgres)
		Password string `yaml:"password"` // password
		DBName   string `yaml:"dbname"`   // database name
		SSLMode  string `yaml:"sslmode"`  // disable (for postgres)
	}

	OpenAIConfig struct {
		APIKey  string `yaml:"api_key"`
		Model   string `yaml:"model"`
		BaseURL string `yaml:"base_url"`
	}
)

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	switch c.Type {
	case "postgres":
		return c.getPostgresDSN()
	case "mysql":
		return c.getMySQLDSN()
	case "sqlite":
		return c.DBName // For SQLite, DBName is the file path
	default:
		return ""
	}
}

// getPostgresDSN returns PostgreSQL connection string
func (c *DatabaseConfig) getPostgresDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
}

// getMySQLDSN returns MySQL connection string
func (c *DatabaseConfig) getMySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.DBName)
}

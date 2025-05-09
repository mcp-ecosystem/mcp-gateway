package config

type (
	StorageConfig struct {
		Type     string            `yaml:"type"`     // disk or db
		Database DatabaseConfig    `yaml:"database"` // database configuration for db type
		Disk     DiskStorageConfig `yaml:"disk"`     // disk configuration for disk type
		API      APIStorageConfig  `yaml:"api"`      // disk configuration for api type
	}

	DiskStorageConfig struct {
		Path string `yaml:"path"` // path for disk storage
	}

	APIStorageConfig struct {
		Url            string `yaml:"url"`            // http url for api
		ConfigJSONPath string `yaml:"configJSONPath"` // configJSONPath for config in http response
		Timeout        int    `yaml:"timeout"`        // timeout(seconds) for http request
	}
)

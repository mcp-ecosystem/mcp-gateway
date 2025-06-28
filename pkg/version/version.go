package version

import (
	_ "embed"
)

//go:embed VERSION
var Version string

// Get returns the current version of the application
func Get() string {
	return Version
}

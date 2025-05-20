package config

import (
	"fmt"
	"strings"
)

// Location represents a configuration location
type Location struct {
	File string
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Message   string
	Locations []Location
}

func (e *ValidationError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Message)
	sb.WriteString("\n\n")
	for _, loc := range e.Locations {
		sb.WriteString("--> ")
		sb.WriteString(loc.File)
		sb.WriteString("\n")
	}
	return sb.String()
}

// ValidateMCPConfig validates a single MCP configuration
func ValidateMCPConfig(cfg *MCPConfig) error {
	return ValidateMCPConfigs([]*MCPConfig{cfg})
}

// ValidateMCPConfigs validates a list of MCP configurations
func ValidateMCPConfigs(configs []*MCPConfig) error {
	var errors []*ValidationError

	// Check for duplicate prefixes
	prefixMap := make(map[string][]Location)
	for _, cfg := range configs {
		for _, router := range cfg.Routers {
			prefix := strings.TrimSuffix(router.Prefix, "/")
			if prefix == "" {
				prefix = "/"
			}
			prefixMap[prefix] = append(prefixMap[prefix], Location{
				File: cfg.Name,
			})
		}
	}

	// Check for duplicate prefixes
	for prefix, locations := range prefixMap {
		if len(locations) > 1 {
			errors = append(errors, &ValidationError{
				Message:   fmt.Sprintf("duplicate prefix %q found in router configurations", prefix),
				Locations: locations,
			})
		}
	}

	// Check for duplicate server names
	serverNameMap := make(map[string][]Location)
	for _, cfg := range configs {
		for _, server := range cfg.Servers {
			serverNameMap[server.Name] = append(serverNameMap[server.Name], Location{
				File: cfg.Name,
			})
		}
	}

	// Check for duplicate server names
	for name, locations := range serverNameMap {
		if len(locations) > 1 {
			errors = append(errors, &ValidationError{
				Message:   fmt.Sprintf("duplicate server name %q found in server configurations", name),
				Locations: locations,
			})
		}
	}

	// Check for duplicate tool names
	toolNameMap := make(map[string][]Location)
	for _, cfg := range configs {
		for _, tool := range cfg.Tools {
			toolNameMap[tool.Name] = append(toolNameMap[tool.Name], Location{
				File: cfg.Name,
			})
		}
	}

	// Check for duplicate tool names
	for name, locations := range toolNameMap {
		if len(locations) > 1 {
			errors = append(errors, &ValidationError{
				Message:   fmt.Sprintf("duplicate tool name %q found in tool configurations", name),
				Locations: locations,
			})
		}
	}

	// Validate router configurations
	for _, cfg := range configs {
		// Build server name map for this config
		serverNames := make(map[string]bool)
		for _, server := range cfg.Servers {
			serverNames[server.Name] = true
		}

		// Also add MCP servers to the server name map
		for _, mcpServer := range cfg.McpServers {
			serverNames[mcpServer.Name] = true
		}

		// Check if all referenced servers exist
		for _, router := range cfg.Routers {
			if !serverNames[router.Server] {
				errors = append(errors, &ValidationError{
					Message: fmt.Sprintf("server %q referenced in router configuration does not exist", router.Server),
					Locations: []Location{{
						File: cfg.Name,
					}},
				})
			}
		}
	}

	if len(errors) > 0 {
		var sb strings.Builder
		for i, err := range errors {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(err.Error())
		}
		return fmt.Errorf("%s", sb.String())
	}

	return nil
}

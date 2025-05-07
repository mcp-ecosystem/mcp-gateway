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
				Message:   fmt.Sprintf("duplicate prefix '%s' found in router configurations", prefix),
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
				Message:   fmt.Sprintf("duplicate server name '%s' found in server configurations", name),
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
				Message:   fmt.Sprintf("duplicate tool name '%s' found in tool configurations", name),
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

		// Check if all referenced servers exist
		for _, router := range cfg.Routers {
			if !serverNames[router.Server] {
				errors = append(errors, &ValidationError{
					Message: fmt.Sprintf("server '%s' referenced in router configuration does not exist", router.Server),
					Locations: []Location{{
						File: cfg.Name,
					}},
				})
			}
		}
	}

	// Check for duplicate stdio names
	stdioNameMap := make(map[string][]Location)
	for _, cfg := range configs {
		for _, stdio := range cfg.StdioConfigs {
			stdioNameMap[stdio.Name] = append(stdioNameMap[stdio.Name], Location{
				File: cfg.Name,
			})
		}
	}

	for name, locations := range stdioNameMap {
		if len(locations) > 1 {
			errors = append(errors, &ValidationError{
				Message:   fmt.Sprintf("duplicate stdio name '%s' found in stdio configurations", name),
				Locations: locations,
			})
		}
	}

	// Check for duplicate sse names
	sseNameMap := make(map[string][]Location)
	for _, cfg := range configs {
		for _, sse := range cfg.SSEConfigs {
			sseNameMap[sse.Name] = append(sseNameMap[sse.Name], Location{
				File: cfg.Name,
			})
		}
	}

	for name, locations := range sseNameMap {
		if len(locations) > 1 {
			errors = append(errors, &ValidationError{
				Message:   fmt.Sprintf("duplicate sse name '%s' found in sse configurations", name),
				Locations: locations,
			})
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
		return fmt.Errorf(sb.String())
	}

	return nil
}

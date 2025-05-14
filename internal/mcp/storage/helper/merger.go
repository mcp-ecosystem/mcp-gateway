package helper

import (
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
)

// MergeConfigs merges all configurations
func MergeConfigs(configs []*config.MCPConfig, items ...*config.MCPConfig) ([]*config.MCPConfig, error) {
	mergedConfig := &config.MCPConfig{}

	for _, cfg := range configs {
		if err := mergeConfig(mergedConfig, cfg); err != nil {
			return nil, err
		}
	}
	for _, cfg := range items {
		if err := mergeConfig(mergedConfig, cfg); err != nil {
			return nil, err
		}
	}

	return []*config.MCPConfig{mergedConfig}, nil
}

// mergeConfig merges two configurations
func mergeConfig(base, override *config.MCPConfig) error {
	// Merge routers
	base.Routers = mergeConfigRouters(base.Routers, override.Routers, !override.DeletedAt.IsZero())

	// Merge servers
	base.Servers = mergeConfigServers(base.Servers, override.Servers, !override.DeletedAt.IsZero())

	// Merge tools
	base.Tools = mergeConfigTools(base.Tools, override.Tools, !override.DeletedAt.IsZero())

	// Merge MCP servers
	base.McpServers = mergeConfigMCPServers(base.McpServers, override.McpServers, !override.DeletedAt.IsZero())

	return nil
}

func mergeConfigMCPServers(base, override []config.MCPServerConfig, deleteMode bool) []config.MCPServerConfig {
	mcpServerMap := make(map[string]config.MCPServerConfig)
	for _, mcpServer := range base {
		mcpServerMap[mcpServer.Name] = mcpServer
	}
	for _, mcpServer := range override {
		if deleteMode {
			delete(mcpServerMap, mcpServer.Name)
		} else {
			mcpServerMap[mcpServer.Name] = mcpServer
		}
	}

	mergedMCPServers := make([]config.MCPServerConfig, 0, len(mcpServerMap))
	for _, mcpServer := range mcpServerMap {
		mergedMCPServers = append(mergedMCPServers, mcpServer)
	}

	return mergedMCPServers
}

func mergeConfigRouters(base, override []config.RouterConfig, deleteMode bool) []config.RouterConfig {
	routerMap := make(map[string]config.RouterConfig)
	for _, router := range base {
		routerMap[router.Server] = router
	}
	for _, router := range override {
		if deleteMode {
			delete(routerMap, router.Server)
		} else {
			routerMap[router.Server] = router
		}
	}

	mergedRouters := make([]config.RouterConfig, 0, len(routerMap))
	for _, router := range routerMap {
		mergedRouters = append(mergedRouters, router)
	}

	return mergedRouters
}

func mergeConfigServers(base, override []config.ServerConfig, deleteMode bool) []config.ServerConfig {
	serverMap := make(map[string]config.ServerConfig)
	for _, server := range base {
		serverMap[server.Name] = server
	}
	for _, server := range override {
		if deleteMode {
			delete(serverMap, server.Name)
		} else {
			serverMap[server.Name] = server
		}
	}

	mergedServers := make([]config.ServerConfig, 0, len(serverMap))
	for _, server := range serverMap {
		mergedServers = append(mergedServers, server)
	}

	return mergedServers
}

func mergeConfigTools(base, override []config.ToolConfig, deleteMode bool) []config.ToolConfig {
	toolMap := make(map[string]config.ToolConfig)
	for _, tool := range base {
		toolMap[tool.Name] = tool
	}
	for _, tool := range override {
		if deleteMode {
			delete(toolMap, tool.Name)
		} else {
			toolMap[tool.Name] = tool
		}
	}

	mergedTools := make([]config.ToolConfig, 0, len(toolMap))
	for _, tool := range toolMap {
		mergedTools = append(mergedTools, tool)
	}

	return mergedTools
}

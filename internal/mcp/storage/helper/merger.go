package helper

import (
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
)

// MergeConfigs merges all configurations
func MergeConfigs(configs []*config.MCPConfig) ([]*config.MCPConfig, error) {
	mergedHTTPConfig := &config.MCPConfig{
		ProtoType: cnst.BackendProtoHttp,
	}
	mergedStdioConfig := &config.MCPConfig{
		ProtoType: cnst.BackendProtoStdio,
	}
	mergedSSEConfig := &config.MCPConfig{
		ProtoType: cnst.BackendProtoSSE,
	}

	for _, cfg := range configs {
		switch cfg.ProtoType {
		case cnst.BackendProtoHttp:
			if err := mergeHTTPConfig(mergedHTTPConfig, cfg); err != nil {
				return nil, err
			}
		case cnst.BackendProtoStdio:
			if err := mergeStdioConfig(mergedStdioConfig, cfg); err != nil {
				return nil, err
			}
		case cnst.BackendProtoSSE:
			if err := mergeSSEConfig(mergedSSEConfig, cfg); err != nil {
				return nil, err
			}
		}
	}

	return []*config.MCPConfig{mergedHTTPConfig, mergedStdioConfig, mergedSSEConfig}, nil
}

// mergeHTTPConfig merges two configurations for HTTP protocol
func mergeHTTPConfig(base, override *config.MCPConfig) error {
	// Merge routers
	base.Routers = mergeConfigRouters(base.Routers, override.Routers)

	// Merge servers
	base.Servers = mergeConfigServers(base.Servers, override.Servers)

	// Merge tools
	base.Tools = mergeConfigTools(base.Tools, override.Tools)

	return nil
}

func mergeStdioConfig(base, override *config.MCPConfig) error {
	// Merge routers
	base.Routers = mergeConfigRouters(base.Routers, override.Routers)

	// Merge servers
	base.Servers = mergeConfigServers(base.Servers, override.Servers)

	// Merge stdio configs
	base.StdioServer = mergeConfigStdio(base.StdioServer, override.StdioServer)

	return nil
}

func mergeSSEConfig(base, override *config.MCPConfig) error {
	// Merge routers
	base.Routers = mergeConfigRouters(base.Routers, override.Routers)

	// Merge servers
	base.Servers = mergeConfigServers(base.Servers, override.Servers)

	// Merge sse configs
	base.SSEServer = mergeConfigSSE(base.SSEServer, override.SSEServer)

	return nil
}

func mergeConfigStdio(base, override config.StdioServerConfig) config.StdioServerConfig {
	return override
}

func mergeConfigSSE(base, override config.SSEServerConfig) config.SSEServerConfig {
	return override
}

func mergeConfigRouters(base, override []config.RouterConfig) []config.RouterConfig {
	routerMap := make(map[string]config.RouterConfig)
	for _, router := range base {
		routerMap[router.Server] = router
	}
	for _, router := range override {
		routerMap[router.Server] = router
	}

	mergedRouters := make([]config.RouterConfig, 0, len(routerMap))
	for _, router := range routerMap {
		mergedRouters = append(mergedRouters, router)
	}

	return mergedRouters
}

func mergeConfigServers(base, override []config.ServerConfig) []config.ServerConfig {
	serverMap := make(map[string]config.ServerConfig)
	for _, server := range base {
		serverMap[server.Name] = server
	}
	for _, server := range override {
		serverMap[server.Name] = server
	}

	mergedServers := make([]config.ServerConfig, 0, len(serverMap))
	for _, server := range serverMap {
		mergedServers = append(mergedServers, server)
	}

	return mergedServers
}

func mergeConfigTools(base, override []config.ToolConfig) []config.ToolConfig {
	toolMap := make(map[string]config.ToolConfig)
	for _, tool := range base {
		toolMap[tool.Name] = tool
	}
	for _, tool := range override {
		toolMap[tool.Name] = tool
	}

	mergedTools := make([]config.ToolConfig, 0, len(toolMap))
	for _, tool := range toolMap {
		mergedTools = append(mergedTools, tool)
	}

	return mergedTools
}

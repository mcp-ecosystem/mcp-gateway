package cnst

// MCPStartupPolicy represents the startup policy for MCP servers
type MCPStartupPolicy string

const (
	// PolicyOnStart represents the policy to connect on server start
	PolicyOnStart MCPStartupPolicy = "onStart"
	// PolicyOnDemand represents the policy to connect when needed
	PolicyOnDemand MCPStartupPolicy = "onDemand"
)

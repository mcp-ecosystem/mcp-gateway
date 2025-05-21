package mcp

// Protocol versions
const (
	ProtocolVersion20250326 = "2025-03-26"
	ProtocolVersion20241105 = "2024-11-05"
	LatestProtocolVersion   = ProtocolVersion20241105
	JSPNRPCVersion          = "2.0"
)

// Methods
const (
	Initialize              = "initialize"
	NotificationInitialized = "notifications/initialized"
	Ping                    = "ping"
	ToolsList               = "tools/list"
	ToolsCall               = "tools/call"
)

// Response
const (
	Accepted = "Accepted"

	NotificationRootsListChanged    = "notifications/roots/list_changed"
	NotificationCancelled           = "notifications/cancelled"
	NotificationProgress            = "notifications/progress"
	NotificationMessage             = "notifications/message"
	NotificationResourceUpdated     = "notifications/resources/updated"
	NotificationResourceListChanged = "notifications/resources/list_changed"
	NotificationToolListChanged     = "notifications/tools/list_changed"
	NotificationPromptListChanged   = "notifications/prompts/list_changed"

	SamplingCreateMessage = "sampling/createMessage"
	LoggingSetLevel       = "logging/setLevel"

	PromptsGet             = "prompts/get"
	PromptsList            = "prompts/list"
	ResourcesList          = "resources/list"
	ResourcesTemplatesList = "resources/templates/list"
	ResourcesRead          = "resources/read"
)

// Error codes for MCP protocol
// Standard JSON-RPC error codes
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// SDKs and applications error codes
const (
	ErrorCodeConnectionClosed = -32000
	ErrorCodeRequestTimeout   = -32001
	ErrorCodeUnauthorized     = -32002
)

const (
	HeaderMcpSessionID = "Mcp-Session-Id"
)

const (
	TextContentType  = "text"
	ImageContentType = "image"
	AudioContentType = "audio"
)

package mcp

import (
	"encoding/json"
)

type (
	JSONRPCBaseResult struct {
		JSONRPC string `json:"jsonrpc"`
		ID      any    `json:"id"`
	}

	// BaseRequestParams represents the base parameters for all requests
	BaseRequestParams struct {
		// Meta information for the request
		Meta RequestMeta `json:"_meta"`
	}

	// RequestMeta represents the meta information for a request
	RequestMeta struct {
		// Progress token for tracking request progress
		// Can be string or number
		ProgressToken any `json:"progressToken"`
	}

	Notification struct {
		Method string             `json:"method"`
		Params NotificationParams `json:"params,omitempty"`
	}

	NotificationParams struct {
		// This parameter name is reserved by MCP to allow clients and
		// servers to attach additional metadata to their notifications.
		Meta map[string]interface{} `json:"_meta,omitempty"`

		// Additional fields can be added to this map
		AdditionalFields map[string]interface{} `json:"-"`
	}

	ToolInputSchema struct {
		Type       string         `json:"type"`
		Properties map[string]any `json:"properties"`
		Required   []string       `json:"required,omitempty"`
		Title      string         `json:"title"`
		Enum       []any          `json:"enum,omitempty"`
	}

	// CallToolParams represents parameters for a tools/call request
	CallToolParams struct {
		BaseRequestParams
		// The name of the tool to call
		Name string `json:"name"`
		// The arguments to pass to the tool
		Arguments json.RawMessage `json:"arguments"`
	}

	AudioContent struct {
		// Must be "audio"
		Type string `json:"type"`
		// The audio data in base64 format
		Data string `json:"data"`
		// The MIME type of the audio. e.g., "audio/wav", "audio/mpeg"
		MimeType string `json:"mimeType"`
	}

	// ImplementationSchema describes the name and version of an MCP implementation
	ImplementationSchema struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	// ClientCapabilitiesSchema represents capabilities a client may support
	ClientCapabilitiesSchema struct {
		Experimental map[string]any        `json:"experimental"`
		Sampling     map[string]any        `json:"sampling"`
		Roots        RootsCapabilitySchema `json:"roots"`
	}

	// RootsCapabilitySchema represents roots-related capabilities
	RootsCapabilitySchema struct {
		ListChanged bool `json:"listChanged"`
	}

	// InitializeRequestParams represents parameters for initialize request
	InitializeRequestParams struct {
		BaseRequestParams
		// The latest version of the Model Context Protocol that the client supports
		ProtocolVersion string `json:"protocolVersion"`
		// Client capabilities
		Capabilities ClientCapabilitiesSchema `json:"capabilities"`
		// Client implementation information
		ClientInfo ImplementationSchema `json:"clientInfo"`
	}

	// InitializeRequestSchema represents an initialize request
	InitializeRequestSchema struct {
		JSONRPCRequest
	}

	// ServerCapabilitiesSchema represents capabilities a server may support
	ServerCapabilitiesSchema struct {
		Experimental ExperimentalCapabilitySchema `json:"experimental"`
		Logging      LoggingCapabilitySchema      `json:"logging"`
		Prompts      PromptsCapabilitySchema      `json:"prompts"`
		Resources    ResourcesCapabilitySchema    `json:"resources"`
		Tools        ToolsCapabilitySchema        `json:"tools"`
	}

	ExperimentalCapabilitySchema struct {
	}

	LoggingCapabilitySchema struct {
	}

	// PromptsCapabilitySchema represents prompts-related capabilities
	PromptsCapabilitySchema struct {
		ListChanged bool `json:"listChanged"`
	}

	// ResourcesCapabilitySchema represents resources-related capabilities
	ResourcesCapabilitySchema struct {
		Subscribe   bool `json:"subscribe"`
		ListChanged bool `json:"listChanged"`
	}

	// ToolsCapabilitySchema represents tools-related capabilities
	ToolsCapabilitySchema struct {
		ListChanged bool `json:"listChanged"`
	}

	InitializedResult struct {
		// The version of the Model Context Protocol that the server wants to use
		ProtocolVersion string `json:"protocolVersion"`
		// Server capabilities
		Capabilities ServerCapabilitiesSchema `json:"capabilities"`
		// Server implementation information
		ServerInfo ImplementationSchema `json:"serverInfo"`
		// Instructions describing how to use the server and its features
		Instructions string `json:"instructions"`
	}

	// InitializedNotification represents an initialized notification
	InitializedNotification struct {
		JSONRPCRequest
	}

	// PingRequest represents a ping request
	PingRequest struct {
		JSONRPCRequest
	}

	JSONRPCErrorSchema struct {
		JSONRPCBaseResult
		Error JSONRPCError `json:"error"`
	}
	// JSONRPCError represents an error in a JSON-RPC response
	JSONRPCError struct {
		// The error type that occurred
		Code int `json:"code"`
		// A short description of the error
		Message string `json:"message"`
		// Additional information about the error
		Data any `json:"data,omitempty"`
	}

	// ServerCapabilities represents capabilities that a server may support. Known
	// capabilities are defined here, in this schema, but this is not a closed set: any
	// server can define its own, additional capabilities.
	ServerCapabilities struct {
		// Experimental, non-standard capabilities that the server supports.
		Experimental map[string]interface{} `json:"experimental,omitempty"`
		// Present if the server supports sending log messages to the client.
		Logging *struct{} `json:"logging,omitempty"`
		// Present if the server offers any prompt templates.
		Prompts *struct {
			// Whether this server supports notifications for changes to the prompt list.
			ListChanged bool `json:"listChanged,omitempty"`
		} `json:"prompts,omitempty"`
		// Present if the server offers any resources to read.
		Resources *struct {
			// Whether this server supports subscribing to resource updates.
			Subscribe bool `json:"subscribe,omitempty"`
			// Whether this server supports notifications for changes to the resource
			// list.
			ListChanged bool `json:"listChanged,omitempty"`
		} `json:"resources,omitempty"`
		// Present if the server offers any tools to call.
		Tools *struct {
			// Whether this server supports notifications for changes to the tool list.
			ListChanged bool `json:"listChanged,omitempty"`
		} `json:"tools,omitempty"`
	}

	// ClientCapabilities represents capabilities a client may support. Known
	// capabilities are defined here, in this schema, but this is not a closed set: any
	// client can define its own, additional capabilities.
	ClientCapabilities struct {
		// Experimental, non-standard capabilities that the client supports.
		Experimental map[string]interface{} `json:"experimental,omitempty"`
		// Present if the client supports listing roots.
		Roots *struct {
			// Whether the client supports notifications for changes to the roots list.
			ListChanged bool `json:"listChanged,omitempty"`
		} `json:"roots,omitempty"`
		// Present if the client supports sampling from an LLM.
		Sampling *struct{} `json:"sampling,omitempty"`
	}
)

// NewInitializeRequest creates a new initialize request
func NewInitializeRequest(id int64, params InitializeRequestParams) InitializeRequestSchema {
	paramsBytes, _ := json.Marshal(params)
	return InitializeRequestSchema{
		JSONRPCRequest: JSONRPCRequest{
			JSONRPC: JSPNRPCVersion,
			ID:      id,
			Method:  Initialize,
			Params:  paramsBytes,
		},
	}
}

// NewPingRequest creates a new ping request
func NewPingRequest(id int64) PingRequest {
	return PingRequest{
		JSONRPCRequest: JSONRPCRequest{
			JSONRPC: JSPNRPCVersion,
			ID:      id,
			Method:  Ping,
		},
	}
}

func NewJSONRPCBaseResult() JSONRPCBaseResult {
	return JSONRPCBaseResult{
		JSONRPC: JSPNRPCVersion,
		ID:      0,
	}
}

func (j JSONRPCBaseResult) WithID(id int) JSONRPCBaseResult {
	j.ID = id
	return j
}

func (t *TextContent) GetType() string {
	return TextContentType
}

func (i *ImageContent) GetType() string {
	return ImageContentType
}

func (i *AudioContent) GetType() string {
	return AudioContentType
}

// NewCallToolResult creates a new CallToolResult
// @param content the content of the result
// @param isError indicates if the result is an error
// @return *CallToolResult the CallToolResult object
func NewCallToolResult(content []Content, isError bool) *CallToolResult {
	return &CallToolResult{
		Content: content,
		IsError: isError,
	}
}

// NewCallToolResultText creates a new CallToolResult with text content
// @param text the text content
// @return *CallToolResult the CallToolResult object with the text content
func NewCallToolResultText(text string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			&TextContent{
				Type: TextContentType,
				Text: text,
			},
		},
		IsError: false,
	}
}

// NewCallToolResultImage  creates a new CallToolResult with an image content
// @param imageData the image data in base64 format
// @param mimeType the MIME type of the image (e.g., "image/png", "image/jpeg")
// @return *CallToolResult the CallToolResult object with the image content
func NewCallToolResultImage(imageData, mimeType string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			&ImageContent{
				Type:     ImageContentType,
				Data:     imageData,
				MIMEType: mimeType,
			},
		},
		IsError: false,
	}
}

// NewCallToolResultAudio creates a new CallToolResult with an audio content
// @param audioData the audio data in base64 format
// @param mimeType the MIME type of the audio (e.g., "audio/wav", "audio/mpeg")
// @return *CallToolResult the CallToolResult object with the audio content
func NewCallToolResultAudio(audioData, mimeType string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			&ImageContent{
				Type:     AudioContentType,
				Data:     audioData,
				MIMEType: mimeType,
			},
		},
		IsError: false,
	}
}

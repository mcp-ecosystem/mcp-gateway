package mcp

// JSONRPC_VERSION is the version of JSON-RPC used by MCP.
const JSONRPC_VERSION = "2.0"

// Cursor is an opaque token used to represent a cursor for pagination.
type Cursor string

// ProgressToken is used to associate progress notifications with the original request.
type ProgressToken interface{}

type Request struct {
	Method string `json:"method"`
	Params struct {
		Meta *struct {
			// If specified, the caller is requesting out-of-band progress
			// notifications for this request (as represented by
			// notifications/progress). The value of this parameter is an
			// opaque token that will be attached to any subsequent
			// notifications. The receiver is not obligated to provide these
			// notifications.
			ProgressToken ProgressToken `json:"progressToken,omitempty"`
		} `json:"_meta,omitempty"`
	} `json:"params,omitempty"`
}

// Implementation describes the name and version of an MCP implementation.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Result struct {
	// This result property is reserved by the protocol to allow clients and
	// servers to attach additional metadata to their responses.
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

type InitializeRequest struct {
	Request
	Params struct {
		// The latest version of the Model Context Protocol that the client supports.
		// The client MAY decide to support older versions as well.
		ProtocolVersion string             `json:"protocolVersion"`
		Capabilities    ClientCapabilities `json:"capabilities"`
		ClientInfo      Implementation     `json:"clientInfo"`
	} `json:"params"`
}

// InitializeResult is sent after receiving an initialize request from the
// client.
type InitializeResult struct {
	Result
	// The version of the Model Context Protocol that the server wants to use.
	// This may not match the version that the client requested. If the client cannot
	// support this version, it MUST disconnect.
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	// Instructions describing how to use the server and its features.
	//
	// This can be used by clients to improve the LLM's understanding of
	// available tools, resources, etc. It can be thought of like a "hint" to the model.
	// For example, this information MAY be added to the system prompt.
	Instructions string `json:"instructions,omitempty"`
}

// JSONRPCNotification represents a notification which does not expect a response.
type JSONRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Notification
}

// CallToolRequest is used by the client to invoke a tool provided by the server.
type CallToolRequest struct {
	Request
	Params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments,omitempty"`
		Meta      *struct {
			// If specified, the caller is requesting out-of-band progress
			// notifications for this request (as represented by
			// notifications/progress). The value of this parameter is an
			// opaque token that will be attached to any subsequent
			// notifications. The receiver is not obligated to provide these
			// notifications.
			ProgressToken ProgressToken `json:"progressToken,omitempty"`
		} `json:"_meta,omitempty"`
	} `json:"params"`
}

// CallToolResult is the server's response to a tool call.
//
// Any errors that originate from the tool SHOULD be reported inside the result
// object, with `isError` set to true, _not_ as an MCP protocol-level error
// response. Otherwise, the LLM would not be able to see that an error occurred
// and self-correct.
//
// However, any errors in _finding_ the tool, an error indicating that the
// server does not support tool calls, or any other exceptional conditions,
// should be reported as an MCP error response.
type CallToolResult struct {
	Result
	Content []Content `json:"content"` // Can be TextContent, ImageContent, or      EmbeddedResource
	// Whether the tool call ended in an error.
	//
	// If not set, this is assumed to be false (the call was successful).
	IsError bool `json:"isError,omitempty"`
}

// Role represents the sender or recipient of messages and data in a
// conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Content interface {
	isContent()
}

// TextContent represents text provided to or from an LLM.
// It must have Type set to "text".
type TextContent struct {
	Annotated
	Type string `json:"type"` // Must be "text"
	// The text content of the message.
	Text string `json:"text"`
}

func (TextContent) isContent() {}

// ImageContent represents an image provided to or from an LLM.
// It must have Type set to "image".
type ImageContent struct {
	Annotated
	Type string `json:"type"` // Must be "image"
	// The base64-encoded image data.
	Data string `json:"data"`
	// The MIME type of the image. Different providers may support different image types.
	MIMEType string `json:"mimeType"`
}

func (ImageContent) isContent() {}

type Annotations struct {
	// Describes who the intended customer of this object or data is.
	//
	// It can include multiple entries to indicate content useful for multiple
	// audiences (e.g., `["user", "assistant"]`).
	Audience []Role `json:"audience,omitempty"`

	// Describes how important this data is for operating the server.
	//
	// A value of 1 means "most important," and indicates that the data is
	// effectively required, while 0 means "least important," and indicates that
	// the data is entirely optional.
	Priority float64 `json:"priority,omitempty"`
}

// Annotated is the base for objects that include optional annotations for the
// client. The client can use annotations to inform how objects are used or
// displayed
type Annotated struct {
	Annotations *Annotations `json:"annotations,omitempty"`
}

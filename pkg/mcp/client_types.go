package mcp

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

// Role represents the sender or recipient of messages and data in a
// conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

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

package session

import (
	"context"
	"time"
)

// Message represents a unified message structure for session communication.
type Message struct {
	Event string // Event type, e.g., "message", "close", "ping"
	Data  []byte // Payload
}

// RequestInfo holds information about the request that created the session.
type RequestInfo struct {
	Headers map[string]string `json:"headers"`
	Query   map[string]string `json:"query"`
	Cookies map[string]string `json:"cookies"`
}

// Meta holds immutable metadata about a session.
type Meta struct {
	ID        string       `json:"id"`         // Unique session ID
	CreatedAt time.Time    `json:"created_at"` // Timestamp of session creation
	Prefix    string       `json:"prefix"`     // Optional namespace or application prefix
	Type      string       `json:"type"`       // Connection type, e.g., "sse", "streamable"
	Request   *RequestInfo `json:"request"`    // Request information
	Extra     []byte       `json:"extra"`      // Optional serialized extra data
}

// Connection represents an active session connection capable of sending messages.
type Connection interface {
	// EventQueue returns a read-only channel where outbound messages are published.
	EventQueue() <-chan *Message

	// Send pushes a message to the session.
	Send(ctx context.Context, msg *Message) error

	// Close gracefully terminates the session connection.
	Close(ctx context.Context) error

	// Meta returns metadata associated with the session.
	Meta() *Meta
}

// Store manages the lifecycle and lookup of active session connections.
type Store interface {
	// Register creates and registers a new session connection.
	Register(ctx context.Context, meta *Meta) (Connection, error)

	// Get retrieves an active session connection by ID.
	Get(ctx context.Context, id string) (Connection, error)

	// Unregister removes a session connection by ID.
	Unregister(ctx context.Context, id string) error

	// List returns all currently active session connections.
	List(ctx context.Context) ([]Connection, error)
}

package session

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// MemoryStore implements Store using in-memory storage
type MemoryStore struct {
	logger *zap.Logger
	mu     sync.RWMutex
	conns  map[string]Connection
}

var _ Store = (*MemoryStore)(nil)

// NewMemoryStore creates a new in-memory session store
func NewMemoryStore(logger *zap.Logger) *MemoryStore {
	return &MemoryStore{
		logger: logger.Named("session.store.memory"),
		conns:  make(map[string]Connection),
	}
}

// Register implements Store.Register
func (s *MemoryStore) Register(_ context.Context, meta *Meta) (Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if connection already exists
	if _, exists := s.conns[meta.ID]; exists {
		return nil, fmt.Errorf("connection already exists: %s", meta.ID)
	}

	// Create new connection
	conn := &MemoryConnection{
		meta:  meta,
		queue: make(chan *Message, 100),
	}

	// Store connection
	s.conns[meta.ID] = conn

	return conn, nil
}

// Get implements Store.Get
func (s *MemoryStore) Get(_ context.Context, id string) (Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conn, ok := s.conns[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return conn, nil
}

// Unregister implements Store.Unregister
func (s *MemoryStore) Unregister(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, ok := s.conns[id]
	if !ok {
		return ErrSessionNotFound
	}

	// Close connection
	if err := conn.Close(context.Background()); err != nil {
		s.logger.Error("failed to close connection",
			zap.String("id", id),
			zap.Error(err))
	}

	// Remove connection
	delete(s.conns, id)
	return nil
}

// List implements Store.List
func (s *MemoryStore) List(_ context.Context) ([]Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conns := make([]Connection, 0, len(s.conns))
	for _, conn := range s.conns {
		conns = append(conns, conn)
	}
	return conns, nil
}

// MemoryConnection implements Connection using in-memory storage
type MemoryConnection struct {
	meta  *Meta
	queue chan *Message
}

var _ Connection = (*MemoryConnection)(nil)

// EventQueue implements Connection.EventQueue
func (c *MemoryConnection) EventQueue() <-chan *Message {
	return c.queue
}

// Send implements Connection.Send
func (c *MemoryConnection) Send(_ context.Context, msg *Message) error {
	select {
	case c.queue <- msg:
		return nil
	default:
		return fmt.Errorf("message queue is full")
	}
}

// Close implements Connection.Close
func (c *MemoryConnection) Close(_ context.Context) error {
	close(c.queue)
	return nil
}

// Meta implements Connection.Meta
func (c *MemoryConnection) Meta() *Meta {
	return c.meta
}

// ErrSessionNotFound is returned when a session is not found
var ErrSessionNotFound = fmt.Errorf("session not found")

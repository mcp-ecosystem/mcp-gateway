package database

import (
	"context"
	"fmt"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Postgres implements the Database interface using PostgreSQL
type Postgres struct {
	db  *gorm.DB
	cfg *config.DatabaseConfig
}

// NewPostgres creates a new Postgres instance
func NewPostgres(cfg *config.DatabaseConfig) (Database, error) {
	db := &Postgres{
		cfg: cfg,
	}

	gormDB, err := gorm.Open(postgres.Open(db.cfg.GetDSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := gormDB.AutoMigrate(&Message{}, &Session{}, &User{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	db.db = gormDB
	return db, nil
}

// Close closes the database connection
func (db *Postgres) Close() error {
	sqlDB, err := db.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// SaveMessage saves a message to the database
func (db *Postgres) SaveMessage(ctx context.Context, message *Message) error {
	return db.db.WithContext(ctx).Create(message).Error
}

// GetMessages retrieves all messages for a session
func (db *Postgres) GetMessages(ctx context.Context, sessionID string) ([]*Message, error) {
	var messages []*Message
	err := db.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("timestamp asc").
		Find(&messages).Error
	return messages, err
}

// GetMessagesWithPagination retrieves messages for a specific session with pagination
func (db *Postgres) GetMessagesWithPagination(ctx context.Context, sessionID string, page, pageSize int) ([]*Message, error) {
	var messages []*Message
	offset := (page - 1) * pageSize
	err := db.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("timestamp asc").
		Offset(offset).
		Limit(pageSize).
		Find(&messages).Error
	return messages, err
}

// CreateSession creates a new session with the given sessionId
func (db *Postgres) CreateSession(ctx context.Context, sessionId string) error {
	session := &Session{
		ID:        sessionId,
		CreatedAt: time.Now(),
	}
	return db.db.WithContext(ctx).Create(session).Error
}

// UpdateSessionTitle updates the title of a session
func (db *Postgres) UpdateSessionTitle(ctx context.Context, sessionID string, title string) error {
	return db.db.WithContext(ctx).
		Model(&Session{}).
		Where("id = ?", sessionID).
		Update("title", title).Error
}

// SessionExists checks if a session exists
func (db *Postgres) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	var count int64
	err := db.db.WithContext(ctx).
		Model(&Session{}).
		Where("id = ?", sessionID).
		Count(&count).Error
	return count > 0, err
}

// GetSessions retrieves all chat sessions with their latest message
func (db *Postgres) GetSessions(ctx context.Context) ([]*Session, error) {
	var sessions []*Session
	err := db.db.WithContext(ctx).
		Order("created_at desc").
		Find(&sessions).Error
	return sessions, err
}

// CreateUser creates a new user
func (db *Postgres) CreateUser(ctx context.Context, user *User) error {
	return db.db.WithContext(ctx).Create(user).Error
}

// GetUserByUsername retrieves a user by username
func (db *Postgres) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := db.db.WithContext(ctx).
		Where("username = ?", username).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates an existing user
func (db *Postgres) UpdateUser(ctx context.Context, user *User) error {
	return db.db.WithContext(ctx).Save(user).Error
}

// DeleteUser deletes a user by ID
func (db *Postgres) DeleteUser(ctx context.Context, id uint) error {
	return db.db.WithContext(ctx).Delete(&User{}, "id = ?", id).Error
}

// ListUsers retrieves all users
func (db *Postgres) ListUsers(ctx context.Context) ([]*User, error) {
	var users []*User
	err := db.db.WithContext(ctx).
		Order("created_at desc").
		Find(&users).Error
	return users, err
}

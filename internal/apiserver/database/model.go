package database

import "time"

// Session represents a chat session
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Title     string    `json:"title"`
}

// Message represents a chat message
type Message struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	Content    string    `json:"content"`
	Sender     string    `json:"sender"`
	Timestamp  time.Time `json:"timestamp"`
	ToolCalls  string    `json:"toolCalls,omitempty"`
	ToolResult string    `json:"toolResult,omitempty"`
}

// UserRole represents the role of a user
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleNormal UserRole = "normal"
)

// User represents an admin user
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"type:varchar(50);uniqueIndex"`
	Password  string    `json:"-" gorm:"not null"` // Password is not exposed in JSON
	Role      UserRole  `json:"role" gorm:"not null;default:'normal'"`
	IsActive  bool      `json:"isActive" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

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

// Tenant represents a tenant in the system
type Tenant struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"type:varchar(50);uniqueIndex"`
	Prefix      string    `json:"prefix" gorm:"type:varchar(50);uniqueIndex"`
	Description string    `json:"description" gorm:"type:varchar(255)"`
	IsActive    bool      `json:"isActive" gorm:"not null;default:true"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// UserTenant represents the relationship between a user and a tenant
type UserTenant struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint      `json:"userId" gorm:"index:idx_user_tenant,unique;not null"`
	TenantID  uint      `json:"tenantId" gorm:"index:idx_user_tenant,unique;not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

package dto

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token string `json:"token"`
}

// InitializeRequest represents an initialization request
type InitializeRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ChangePasswordRequest represents a request to change password
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// ChangePasswordResponse represents a response to change password
type ChangePasswordResponse struct {
	Success bool `json:"success"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=admin normal"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role,omitempty" binding:"omitempty,oneof=admin normal"`
	IsActive *bool  `json:"isActive,omitempty"`
}

// UserInfo represents the user information stored in the context
type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

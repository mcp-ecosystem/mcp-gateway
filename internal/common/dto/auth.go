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
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Role      string `json:"role" binding:"required,oneof=admin normal"`
	TenantIDs []uint `json:"tenantIds,omitempty"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password,omitempty"`
	Role      string `json:"role,omitempty" binding:"omitempty,oneof=admin normal"`
	IsActive  *bool  `json:"isActive,omitempty"`
	TenantIDs []uint `json:"tenantIds,omitempty"`
}

// UserInfo represents the user information stored in the context
type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// CreateTenantRequest represents a request to create a new tenant
type CreateTenantRequest struct {
	Name        string `json:"name" binding:"required"`
	Prefix      string `json:"prefix" binding:"required"`
	Description string `json:"description"`
}

// UpdateTenantRequest represents a request to update a tenant
type UpdateTenantRequest struct {
	Name        string `json:"name" binding:"required"`
	Prefix      string `json:"prefix,omitempty"`
	Description string `json:"description,omitempty"`
	IsActive    *bool  `json:"isActive,omitempty"`
}

// TenantResponse represents a tenant information response
type TenantResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Prefix      string `json:"prefix"`
	Description string `json:"description"`
	IsActive    bool   `json:"isActive"`
}

// UserResponse represents a user with their associated tenants
type UserResponse struct {
	ID       uint              `json:"id"`
	Username string            `json:"username"`
	Role     string            `json:"role"`
	IsActive bool              `json:"isActive"`
	Tenants  []*TenantResponse `json:"tenants,omitempty"`
}

// UserTenantRequest represents a request to add or remove tenant associations
type UserTenantRequest struct {
	UserID    uint   `json:"userId" binding:"required"`
	TenantIDs []uint `json:"tenantIds" binding:"required"`
}

// TenantUserRequest represents a request to add or remove user associations
type TenantUserRequest struct {
	TenantID uint   `json:"tenantId" binding:"required"`
	UserIDs  []uint `json:"userIds" binding:"required"`
}

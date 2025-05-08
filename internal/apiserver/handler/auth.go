package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/auth/jwt"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/dto"
	"golang.org/x/crypto/bcrypt"
)

// Handler represents the authentication handler
type Handler struct {
	db         database.Database
	jwtService *jwt.Service
	cfg        *config.MCPGatewayConfig
}

// NewHandler creates a new authentication handler
func NewHandler(db database.Database, jwtService *jwt.Service, cfg *config.MCPGatewayConfig) *Handler {
	return &Handler{
		db:         db,
		jwtService: jwtService,
		cfg:        cfg,
	}
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from database
	user, err := h.db.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "User is disabled"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user.ID, user.Username, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Store user info in context for future requests
	userInfo := &dto.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Role:     string(user.Role),
	}
	c.Set("user", userInfo)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  userInfo,
	})
}

// ChangePassword handles password change requests
func (h *Handler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Get the user from the context
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	// Get the user from the database
	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Compare the old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid old password"})
		return
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Update the user's password
	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()
	if err := h.db.UpdateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dto.ChangePasswordResponse{Success: true})
}

// AdminAuthMiddleware creates a middleware that checks if the user has admin role
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		jwtClaims, ok := claims.(*jwt.Claims)
		if !ok || jwtClaims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Only administrators can access this resource"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ListUsers handles listing all users
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.db.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// CreateUser handles user creation
func (h *Handler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate request
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create user
	newUser := &database.User{
		Username:  req.Username,
		Password:  string(hashedPassword),
		Role:      database.UserRole(req.Role),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.db.CreateUser(c.Request.Context(), newUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": newUser.ID})
}

// UpdateUser handles user updates
func (h *Handler) UpdateUser(c *gin.Context) {
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the user from the database
	existingUser, err := h.db.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update user fields
	if req.Role != "" {
		existingUser.Role = database.UserRole(req.Role)
	}
	if req.IsActive != nil {
		existingUser.IsActive = *req.IsActive
	}
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		existingUser.Password = string(hashedPassword)
	}
	existingUser.UpdatedAt = time.Now()

	if err := h.db.UpdateUser(c.Request.Context(), existingUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser handles user deletion
func (h *Handler) DeleteUser(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
		return
	}

	// Get the user from the database
	existingUser, err := h.db.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Delete the user
	if err := h.db.DeleteUser(c.Request.Context(), existingUser.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// GetUserInfo handles getting current user info
func (h *Handler) GetUserInfo(c *gin.Context) {
	// Get the user from the context
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	// Get the user from the database
	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
		"status":   user.IsActive,
	})
}

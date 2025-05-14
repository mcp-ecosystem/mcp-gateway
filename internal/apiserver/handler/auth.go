package handler

import (
	"net/http"
	"time"

	"context"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/auth/jwt"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/dto"
	"github.com/mcp-ecosystem/mcp-gateway/internal/i18n"
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
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", err.Error()))
		return
	}

	// Get user from database
	user, err := h.db.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrorInvalidCredentials)
		return
	}

	// Check if user is active
	if !user.IsActive {
		i18n.RespondWithError(c, i18n.ErrorUserDisabled)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		i18n.RespondWithError(c, i18n.ErrorInvalidCredentials)
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user.ID, user.Username, string(user.Role))
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	// Store user info in context for future requests
	userInfo := &dto.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Role:     string(user.Role),
	}
	c.Set("user", userInfo)

	i18n.Success(i18n.SuccessLogin).
		With("token", token).
		With("user", userInfo).
		Send(c)
}

// ChangePassword handles password change requests
func (h *Handler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", err.Error()))
		return
	}

	// Get the user from the context
	claims, exists := c.Get("claims")
	if !exists {
		i18n.RespondWithError(c, i18n.ErrUnauthorized)
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	// Get the user from the database
	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	// Compare the old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		i18n.RespondWithError(c, i18n.ErrorInvalidOldPassword)
		return
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	// Update the user's password
	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()
	if err := h.db.UpdateUser(c.Request.Context(), user); err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	i18n.Success(i18n.SuccessPasswordChanged).With("success", true).Send(c)
}

// AdminAuthMiddleware creates a middleware that checks if the user has admin role
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			i18n.RespondWithError(c, i18n.ErrUnauthorized)
			c.Abort()
			return
		}

		jwtClaims, ok := claims.(*jwt.Claims)
		if !ok || jwtClaims.Role != "admin" {
			i18n.RespondWithError(c, i18n.ErrForbidden.WithParam("Reason", "Only administrators can access this resource"))
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
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	c.JSON(http.StatusOK, users)
}

// CreateUser handles user creation
func (h *Handler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", err.Error()))
		return
	}

	// Validate request
	if req.Username == "" || req.Password == "" {
		i18n.RespondWithError(c, i18n.ErrorUserNamePasswordRequired)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	// Create user with transaction to handle tenant associations
	var userID uint
	err = h.db.Transaction(c.Request.Context(), func(ctx context.Context) error {
		// Create user
		newUser := &database.User{
			Username:  req.Username,
			Password:  string(hashedPassword),
			Role:      database.UserRole(req.Role),
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := h.db.CreateUser(ctx, newUser); err != nil {
			return err
		}

		userID = newUser.ID

		// Associate user with tenants if provided
		if len(req.TenantIDs) > 0 {
			for _, tenantID := range req.TenantIDs {
				if err := h.db.AddUserToTenant(ctx, newUser.ID, tenantID); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	i18n.Created(i18n.SuccessUserCreated).With("id", userID).Send(c)
}

// UpdateUser handles user updates
func (h *Handler) UpdateUser(c *gin.Context) {
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", err.Error()))
		return
	}

	// Get the user from the database
	existingUser, err := h.db.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrorUserNotFound.WithParam("Username", req.Username))
		return
	}

	// Update user with transaction to handle tenant associations
	err = h.db.Transaction(c.Request.Context(), func(ctx context.Context) error {
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
				return err
			}
			existingUser.Password = string(hashedPassword)
		}
		existingUser.UpdatedAt = time.Now()

		if err := h.db.UpdateUser(ctx, existingUser); err != nil {
			return err
		}

		// Update tenant associations if provided
		if req.TenantIDs != nil {
			// Get existing tenant IDs for the user
			existingTenants, err := h.db.GetUserTenants(ctx, existingUser.ID)
			if err != nil {
				return err
			}

			// Create a map of existing tenant IDs for easy lookup
			existingTenantIDs := make(map[uint]bool)
			for _, tenant := range existingTenants {
				existingTenantIDs[tenant.ID] = true
			}

			// Create a map of new tenant IDs for easy lookup
			newTenantIDs := make(map[uint]bool)
			for _, id := range req.TenantIDs {
				newTenantIDs[id] = true
			}

			// Remove associations that no longer exist in the request
			for _, tenant := range existingTenants {
				if !newTenantIDs[tenant.ID] {
					if err := h.db.RemoveUserFromTenant(ctx, existingUser.ID, tenant.ID); err != nil {
						return err
					}
				}
			}

			// Add new associations that don't exist yet
			for _, id := range req.TenantIDs {
				if !existingTenantIDs[id] {
					if err := h.db.AddUserToTenant(ctx, existingUser.ID, id); err != nil {
						return err
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", err.Error()))
		return
	}

	i18n.Success(i18n.SuccessUserUpdated).Send(c)
}

// DeleteUser handles user deletion
func (h *Handler) DeleteUser(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Username is required"))
		return
	}

	// Get the user from the database
	existingUser, err := h.db.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrorUserNotFound.WithParam("Username", username))
		return
	}

	// Delete user and related tenant associations in a transaction
	err = h.db.Transaction(c.Request.Context(), func(ctx context.Context) error {
		// First delete user-tenant associations
		if err := h.db.DeleteUserTenants(ctx, existingUser.ID); err != nil {
			return err
		}

		// Then delete the user itself
		if err := h.db.DeleteUser(ctx, existingUser.ID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", err.Error()))
		return
	}

	i18n.Success(i18n.SuccessUserDeleted).Send(c)
}

// GetUserInfo handles getting current user info
func (h *Handler) GetUserInfo(c *gin.Context) {
	// Get the user from the context
	claims, exists := c.Get("claims")
	if !exists {
		i18n.RespondWithError(c, i18n.ErrUnauthorized)
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	// Get the user from the database
	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	var tenants []*database.Tenant
	var err2 error

	// If user is admin, get all tenants
	if user.Role == database.RoleAdmin {
		tenants, err2 = h.db.ListTenants(c.Request.Context())
	} else {
		// Non-admin users only get assigned tenants
		tenants, err2 = h.db.GetUserTenants(c.Request.Context(), user.ID)
	}

	if err2 != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get tenants"))
		return
	}

	// Convert tenants to tenant responses
	tenantResponses := make([]*dto.TenantResponse, len(tenants))
	for i, tenant := range tenants {
		tenantResponses[i] = &dto.TenantResponse{
			ID:          tenant.ID,
			Name:        tenant.Name,
			Prefix:      tenant.Prefix,
			Description: tenant.Description,
			IsActive:    tenant.IsActive,
		}
	}

	i18n.Success(i18n.SuccessUserInfo).
		With("id", user.ID).
		With("username", user.Username).
		With("role", user.Role).
		With("isActive", user.IsActive).
		With("tenants", tenantResponses).
		Send(c)
}

// GetUserWithTenants gets a user with their associated tenants
func (h *Handler) GetUserWithTenants(c *gin.Context) {
	// Get current logged-in user information for permission checking
	claims, exists := c.Get("claims")
	if !exists {
		i18n.RespondWithError(c, i18n.ErrUnauthorized)
		return
	}
	currentUserClaims := claims.(*jwt.Claims)

	// Check if path parameter exists, if not use the current logged-in user
	username := c.Param("username")
	useCurrentUser := username == ""

	// If no username parameter is provided, use the current logged-in user
	if useCurrentUser {
		username = currentUserClaims.Username
	} else {
		// Only administrators can view information of other users
		if currentUserClaims.Role != "admin" && username != currentUserClaims.Username {
			i18n.RespondWithError(c, i18n.ErrForbidden.WithParam("Reason", "Only administrators can access other users' information"))
			return
		}
	}

	// Get user from database
	user, err := h.db.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrorUserNotFound.WithParam("Username", username))
		return
	}

	var tenants []*database.Tenant
	var err2 error

	// If user is admin, get all tenants
	if user.Role == database.RoleAdmin {
		tenants, err2 = h.db.ListTenants(c.Request.Context())
	} else {
		// Non-admin users only get assigned tenants
		tenants, err2 = h.db.GetUserTenants(c.Request.Context(), user.ID)
	}

	if err2 != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get user tenants"))
		return
	}

	// Convert tenants to tenant responses
	tenantResponses := make([]*dto.TenantResponse, len(tenants))
	for i, tenant := range tenants {
		tenantResponses[i] = &dto.TenantResponse{
			ID:          tenant.ID,
			Name:        tenant.Name,
			Prefix:      tenant.Prefix,
			Description: tenant.Description,
			IsActive:    tenant.IsActive,
		}
	}

	// Create user response with tenants
	userResponse := &dto.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Role:     string(user.Role),
		IsActive: user.IsActive,
		Tenants:  tenantResponses,
	}

	i18n.Success(i18n.SuccessUserWithTenants).WithPayload(userResponse).Send(c)
}

// UpdateUserTenants updates the tenant associations for a user
func (h *Handler) UpdateUserTenants(c *gin.Context) {
	var req dto.UserTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", err.Error()))
		return
	}

	// Perform the update in a transaction
	err := h.db.Transaction(c.Request.Context(), func(ctx context.Context) error {
		// Get existing tenant IDs for the user
		existingTenants, err := h.db.GetUserTenants(ctx, req.UserID)
		if err != nil {
			return err
		}

		// Create a map of existing tenant IDs for easy lookup
		existingTenantIDs := make(map[uint]bool)
		for _, tenant := range existingTenants {
			existingTenantIDs[tenant.ID] = true
		}

		// Create a map of new tenant IDs for easy lookup
		newTenantIDs := make(map[uint]bool)
		for _, id := range req.TenantIDs {
			newTenantIDs[id] = true
		}

		// Remove associations that no longer exist in the request
		for _, tenant := range existingTenants {
			if !newTenantIDs[tenant.ID] {
				if err := h.db.RemoveUserFromTenant(ctx, req.UserID, tenant.ID); err != nil {
					return err
				}
			}
		}

		// Add new associations that don't exist yet
		for _, id := range req.TenantIDs {
			if !existingTenantIDs[id] {
				if err := h.db.AddUserToTenant(ctx, req.UserID, id); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", err.Error()))
		return
	}

	i18n.Success(i18n.SuccessUserTenantsUpdated).Send(c)
}

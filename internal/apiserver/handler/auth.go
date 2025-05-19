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
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Handler represents the authentication handler
type Handler struct {
	db         database.Database
	jwtService *jwt.Service
	cfg        *config.MCPGatewayConfig
	logger     *zap.Logger
}

// NewHandler creates a new authentication handler
func NewHandler(db database.Database, jwtService *jwt.Service, cfg *config.MCPGatewayConfig, logger *zap.Logger) *Handler {
	return &Handler{
		db:         db,
		jwtService: jwtService,
		cfg:        cfg,
		logger:     logger.Named("apiserver.handler.auth"),
	}
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid login request format",
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", err.Error()))
		return
	}

	h.logger.Info("processing login request",
		zap.String("username", req.Username),
		zap.String("remote_addr", c.ClientIP()))

	user, err := h.db.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		h.logger.Warn("login failed: user not found",
			zap.String("username", req.Username),
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrorInvalidCredentials)
		return
	}

	if !user.IsActive {
		h.logger.Warn("login attempt for disabled user",
			zap.String("username", req.Username),
			zap.Uint("user_id", user.ID),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrorUserDisabled)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.logger.Warn("login failed: invalid password",
			zap.String("username", req.Username),
			zap.Uint("user_id", user.ID),
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrorInvalidCredentials)
		return
	}

	token, err := h.jwtService.GenerateToken(user.ID, user.Username, string(user.Role))
	if err != nil {
		h.logger.Error("failed to generate JWT token",
			zap.Error(err),
			zap.String("username", req.Username),
			zap.Uint("user_id", user.ID),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	h.logger.Info("user logged in successfully",
		zap.String("username", req.Username),
		zap.Uint("user_id", user.ID),
		zap.String("role", string(user.Role)),
		zap.String("remote_addr", c.ClientIP()))

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
		h.logger.Warn("invalid change password request format",
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", err.Error()))
		return
	}

	claims, exists := c.Get("claims")
	if !exists {
		h.logger.Warn("unauthorized change password attempt",
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrUnauthorized)
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	h.logger.Info("processing change password request",
		zap.String("username", jwtClaims.Username),
		zap.String("remote_addr", c.ClientIP()))

	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		h.logger.Error("failed to get user for password change",
			zap.Error(err),
			zap.String("username", jwtClaims.Username),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		h.logger.Warn("invalid old password in change password request",
			zap.String("username", jwtClaims.Username),
			zap.Uint("user_id", user.ID),
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrorInvalidOldPassword)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("failed to hash new password",
			zap.Error(err),
			zap.String("username", jwtClaims.Username),
			zap.Uint("user_id", user.ID),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()
	if err := h.db.UpdateUser(c.Request.Context(), user); err != nil {
		h.logger.Error("failed to update user password in database",
			zap.Error(err),
			zap.String("username", jwtClaims.Username),
			zap.Uint("user_id", user.ID),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	h.logger.Info("password changed successfully",
		zap.String("username", jwtClaims.Username),
		zap.Uint("user_id", user.ID),
		zap.String("remote_addr", c.ClientIP()))

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
	claims, exists := c.Get("claims")
	if !exists || claims == nil {
		h.logger.Warn("unauthorized list users attempt",
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrUnauthorized)
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	h.logger.Info("listing all users",
		zap.String("username", jwtClaims.Username),
		zap.String("remote_addr", c.ClientIP()))

	users, err := h.db.ListUsers(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to list users from database",
			zap.Error(err),
			zap.String("username", jwtClaims.Username),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	h.logger.Debug("users list retrieved successfully",
		zap.Int("user_count", len(users)),
		zap.String("username", jwtClaims.Username),
		zap.String("remote_addr", c.ClientIP()))

	c.JSON(http.StatusOK, users)
}

// CreateUser handles user creation
func (h *Handler) CreateUser(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists || claims == nil {
		h.logger.Warn("unauthorized create user attempt",
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrUnauthorized)
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid create user request format",
			zap.Error(err),
			zap.String("admin", jwtClaims.Username),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", err.Error()))
		return
	}

	h.logger.Info("processing user creation request",
		zap.String("new_username", req.Username),
		zap.String("role", req.Role),
		zap.Any("tenant_ids", req.TenantIDs),
		zap.String("admin", jwtClaims.Username),
		zap.String("remote_addr", c.ClientIP()))

	if req.Username == "" || req.Password == "" {
		h.logger.Warn("create user missing required fields",
			zap.String("username", req.Username),
			zap.String("admin", jwtClaims.Username),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrorUserNamePasswordRequired)
		return
	}

	existingUser, err := h.db.GetUserByUsername(c.Request.Context(), req.Username)
	if err == nil && existingUser != nil {
		h.logger.Warn("attempt to create user with existing username",
			zap.String("username", req.Username),
			zap.String("admin", jwtClaims.Username),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "User already exists"))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("failed to hash password for new user",
			zap.Error(err),
			zap.String("username", req.Username),
			zap.String("admin", jwtClaims.Username),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	h.logger.Debug("password hashed successfully for new user",
		zap.String("username", req.Username),
		zap.String("admin", jwtClaims.Username))

	var userID uint
	err = h.db.Transaction(c.Request.Context(), func(ctx context.Context) error {
		newUser := &database.User{
			Username:  req.Username,
			Password:  string(hashedPassword),
			Role:      database.UserRole(req.Role),
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := h.db.CreateUser(ctx, newUser); err != nil {
			h.logger.Error("failed to create user in database",
				zap.Error(err),
				zap.String("username", req.Username),
				zap.String("admin", jwtClaims.Username))
			return err
		}

		userID = newUser.ID
		h.logger.Debug("user created successfully",
			zap.String("username", req.Username),
			zap.Uint("user_id", userID),
			zap.String("admin", jwtClaims.Username))

		if len(req.TenantIDs) > 0 {
			for _, tenantID := range req.TenantIDs {
				if err := h.db.AddUserToTenant(ctx, newUser.ID, tenantID); err != nil {
					h.logger.Error("failed to add user to tenant",
						zap.Error(err),
						zap.String("username", req.Username),
						zap.Uint("user_id", userID),
						zap.Uint("tenant_id", tenantID),
						zap.String("admin", jwtClaims.Username))
					return err
				}
				h.logger.Debug("user added to tenant",
					zap.String("username", req.Username),
					zap.Uint("user_id", userID),
					zap.Uint("tenant_id", tenantID),
					zap.String("admin", jwtClaims.Username))
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

	existingUser, err := h.db.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrorUserNotFound.WithParam("Username", req.Username))
		return
	}

	err = h.db.Transaction(c.Request.Context(), func(ctx context.Context) error {
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

		if req.TenantIDs != nil {
			existingTenants, err := h.db.GetUserTenants(ctx, existingUser.ID)
			if err != nil {
				return err
			}

			existingTenantIDs := make(map[uint]bool)
			for _, tenant := range existingTenants {
				existingTenantIDs[tenant.ID] = true
			}

			newTenantIDs := make(map[uint]bool)
			for _, id := range req.TenantIDs {
				newTenantIDs[id] = true
			}

			for _, tenant := range existingTenants {
				if !newTenantIDs[tenant.ID] {
					if err := h.db.RemoveUserFromTenant(ctx, existingUser.ID, tenant.ID); err != nil {
						return err
					}
				}
			}

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

	existingUser, err := h.db.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrorUserNotFound.WithParam("Username", username))
		return
	}

	err = h.db.Transaction(c.Request.Context(), func(ctx context.Context) error {
		if err := h.db.DeleteUserTenants(ctx, existingUser.ID); err != nil {
			return err
		}

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
	claims, exists := c.Get("claims")
	if !exists {
		i18n.RespondWithError(c, i18n.ErrUnauthorized)
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer)
		return
	}

	var tenants []*database.Tenant
	var err2 error

	if user.Role == database.RoleAdmin {
		tenants, err2 = h.db.ListTenants(c.Request.Context())
	} else {
		tenants, err2 = h.db.GetUserTenants(c.Request.Context(), user.ID)
	}

	if err2 != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get tenants"))
		return
	}

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
	claims, exists := c.Get("claims")
	if !exists {
		i18n.RespondWithError(c, i18n.ErrUnauthorized)
		return
	}
	currentUserClaims := claims.(*jwt.Claims)

	username := c.Param("username")
	useCurrentUser := username == ""

	if useCurrentUser {
		username = currentUserClaims.Username
	} else {
		if currentUserClaims.Role != "admin" && username != currentUserClaims.Username {
			i18n.RespondWithError(c, i18n.ErrForbidden.WithParam("Reason", "Only administrators can access other users' information"))
			return
		}
	}

	user, err := h.db.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrorUserNotFound.WithParam("Username", username))
		return
	}

	var tenants []*database.Tenant
	var err2 error

	if user.Role == database.RoleAdmin {
		tenants, err2 = h.db.ListTenants(c.Request.Context())
	} else {
		tenants, err2 = h.db.GetUserTenants(c.Request.Context(), user.ID)
	}

	if err2 != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get user tenants"))
		return
	}

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

	err := h.db.Transaction(c.Request.Context(), func(ctx context.Context) error {
		existingTenants, err := h.db.GetUserTenants(ctx, req.UserID)
		if err != nil {
			return err
		}

		existingTenantIDs := make(map[uint]bool)
		for _, tenant := range existingTenants {
			existingTenantIDs[tenant.ID] = true
		}

		newTenantIDs := make(map[uint]bool)
		for _, id := range req.TenantIDs {
			newTenantIDs[id] = true
		}

		for _, tenant := range existingTenants {
			if !newTenantIDs[tenant.ID] {
				if err := h.db.RemoveUserFromTenant(ctx, req.UserID, tenant.ID); err != nil {
					return err
				}
			}
		}

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

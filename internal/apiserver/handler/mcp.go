package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/auth/jwt"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/dto"
	"github.com/mcp-ecosystem/mcp-gateway/internal/i18n"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage/notifier"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type MCP struct {
	db       database.Database
	store    storage.Store
	notifier notifier.Notifier
	logger   *zap.Logger
}

func NewMCP(db database.Database, store storage.Store, ntf notifier.Notifier, logger *zap.Logger) *MCP {
	return &MCP{
		db:       db,
		store:    store,
		notifier: ntf,
		logger:   logger,
	}
}

// checkTenantPermission checks if the user has permission to access the specified tenant and
// verifies that all router prefixes start with the tenant prefix as a complete path segment
func (h *MCP) checkTenantPermission(c *gin.Context, tenantName string, cfg *config.MCPConfig) (*database.Tenant, error) {
	h.logger.Debug("checking tenant permission",
		zap.String("tenant_name", tenantName))

	// Check if tenant name is empty
	if tenantName == "" {
		h.logger.Warn("tenant name is empty")
		return nil, i18n.ErrorTenantNameRequired
	}

	// Get user authentication information
	claims, exists := c.Get("claims")
	if !exists {
		h.logger.Warn("missing JWT claims in context")
		return nil, i18n.ErrUnauthorized
	}
	jwtClaims := claims.(*jwt.Claims)

	// Get user information
	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		h.logger.Error("failed to get user info",
			zap.String("username", jwtClaims.Username),
			zap.Error(err))
		return nil, i18n.ErrInternalServer.WithParam("Reason", "Failed to get user info: "+err.Error())
	}

	// Get tenant information
	tenant, err := h.db.GetTenantByName(c.Request.Context(), tenantName)
	if err != nil {
		h.logger.Warn("tenant not found",
			zap.String("tenant_name", tenantName),
			zap.Error(err))
		return nil, i18n.ErrorTenantNotFound.WithParam("Name", tenantName)
	}

	// Normalize tenant prefix
	tenantPrefix := tenant.Prefix
	if !strings.HasPrefix(tenantPrefix, "/") {
		tenantPrefix = "/" + tenantPrefix
	}
	tenantPrefix = strings.TrimSuffix(tenantPrefix, "/")

	// Check if all router prefixes start with tenant prefix
	for _, router := range cfg.Routers {
		// Normalize router prefix
		routerPrefix := router.Prefix
		if !strings.HasPrefix(routerPrefix, "/") {
			routerPrefix = "/" + routerPrefix
		}
		routerPrefix = strings.TrimSuffix(routerPrefix, "/")

		// Allow exact match
		if routerPrefix == tenantPrefix {
			continue
		}

		// Must start with tenant prefix followed by a path separator
		if !strings.HasPrefix(routerPrefix, tenantPrefix+"/") {
			h.logger.Warn("router prefix validation failed",
				zap.String("router_prefix", routerPrefix),
				zap.String("tenant_prefix", tenantPrefix))
			return nil, i18n.ErrorRouterPrefixError
		}
	}

	// Check user permission if not admin
	if user.Role != database.RoleAdmin {
		userTenants, err := h.db.GetUserTenants(c.Request.Context(), user.ID)
		if err != nil {
			h.logger.Error("failed to get user tenants",
				zap.Uint("user_id", user.ID),
				zap.Error(err))
			return nil, i18n.ErrInternalServer.WithParam("Reason", "Failed to get user tenants: "+err.Error())
		}

		allowed := false
		for _, userTenant := range userTenants {
			if userTenant.ID == tenant.ID {
				allowed = true
				break
			}
		}

		if !allowed {
			h.logger.Warn("user lacks permission for tenant",
				zap.Uint("user_id", user.ID),
				zap.Uint("tenant_id", tenant.ID))
			return nil, i18n.ErrorTenantPermissionError
		}
	}

	h.logger.Debug("tenant permission check passed",
		zap.String("tenant_name", tenantName),
		zap.Uint("tenant_id", tenant.ID))
	return tenant, nil
}

func (h *MCP) HandleMCPServerUpdate(c *gin.Context) {
	// Get the server name from path parameter instead of query parameter
	name := c.Param("name")
	if name == "" {
		h.logger.Warn("MCP server name required but missing")
		i18n.RespondWithError(c, i18n.ErrorMCPServerNameRequired)
		return
	}

	h.logger.Info("handling MCP server update request",
		zap.String("server_name", name))

	// Read the raw YAML content from request body
	content, err := c.GetRawData()
	if err != nil {
		h.logger.Error("failed to read request body", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Failed to read request body: "+err.Error()))
		return
	}

	// Validate the YAML content
	var cfg config.MCPConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		h.logger.Error("invalid YAML content", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Invalid YAML content: "+err.Error()))
		return
	}

	// Get existing server
	oldCfg, err := h.store.Get(c.Request.Context(), name)
	if err != nil {
		h.logger.Error("MCP server not found",
			zap.String("server_name", name),
			zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrorMCPServerNotFound.WithParam("Name", name))
		return
	}

	if oldCfg.Name != cfg.Name {
		h.logger.Warn("server name mismatch",
			zap.String("param_name", name),
			zap.String("config_name", cfg.Name))
		i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Server name in configuration must match name parameter"))
		return
	}

	_, err = h.checkTenantPermission(c, cfg.Tenant, &cfg)
	if err != nil {
		h.logger.Warn("tenant permission check failed",
			zap.String("tenant", cfg.Tenant),
			zap.Error(err))
		i18n.RespondWithError(c, err)
		return
	}

	// Get all existing configurations
	configs, err := h.store.List(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to get existing configurations", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get existing configurations: "+err.Error()))
		return
	}

	// Merge the new configuration with existing configs
	configs = config.MergeConfigs(configs, &cfg)

	// Validate all configurations
	if err := config.ValidateMCPConfigs(configs); err != nil {
		var validationErr *config.ValidationError
		if errors.As(err, &validationErr) {
			h.logger.Error("configuration validation failed",
				zap.String("validation_error", validationErr.Error()))
			i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Configuration validation failed: "+validationErr.Error()))
		} else {
			h.logger.Error("failed to validate configurations", zap.Error(err))
			i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Failed to validate configurations: "+err.Error()))
		}
		return
	}

	if err := h.store.Update(c.Request.Context(), &cfg); err != nil {
		h.logger.Error("failed to update MCP server",
			zap.String("server_name", name),
			zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to update MCP server: "+err.Error()))
		return
	}

	// Send reload signal to gateway using notifier
	if err := h.notifier.NotifyUpdate(c.Request.Context(), &cfg); err != nil {
		h.logger.Error("failed to reload gateway", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to reload gateway: "+err.Error()))
		return
	}

	h.logger.Info("MCP server updated successfully",
		zap.String("server_name", name))
	i18n.Success(i18n.SuccessMCPServerUpdated).With("status", "success").Send(c)
}

func (h *MCP) HandleListMCPServers(c *gin.Context) {
	h.logger.Info("handling list MCP servers request")

	tenantIDStr := c.Query("tenantId")
	var tenantID uint
	if tenantIDStr != "" {
		tid, err := strconv.ParseUint(tenantIDStr, 10, 32)
		if err != nil {
			h.logger.Warn("invalid tenantId parameter",
				zap.String("tenantId", tenantIDStr))
			i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Invalid tenantId parameter"))
			return
		}
		tenantID = uint(tid)
		h.logger.Debug("filtering by tenant ID", zap.Uint("tenant_id", tenantID))
	}

	claims, exists := c.Get("claims")
	if !exists {
		h.logger.Warn("missing JWT claims in context")
		i18n.RespondWithError(c, i18n.ErrUnauthorized)
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		h.logger.Error("failed to get user info",
			zap.String("username", jwtClaims.Username),
			zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get user info: "+err.Error()))
		return
	}

	servers, err := h.store.List(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to get MCP servers", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get MCP servers: "+err.Error()))
		return
	}

	if user.Role != database.RoleAdmin && tenantID > 0 {
		userTenants, err := h.db.GetUserTenants(c.Request.Context(), user.ID)
		if err != nil {
			h.logger.Error("failed to get user tenants",
				zap.Uint("user_id", user.ID),
				zap.Error(err))
			i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get user tenants: "+err.Error()))
			return
		}

		hasPermission := false
		for _, tenant := range userTenants {
			if tenant.ID == tenantID {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			h.logger.Warn("user lacks permission for tenant",
				zap.Uint("user_id", user.ID),
				zap.Uint("tenant_id", tenantID))
			i18n.RespondWithError(c, i18n.ErrorTenantPermissionError)
			return
		}
	}

	var filteredServers []*config.MCPConfig
	if tenantID > 0 {
		tenant, err := h.db.GetTenantByID(c.Request.Context(), tenantID)
		if err != nil {
			h.logger.Error("tenant not found",
				zap.Uint("tenant_id", tenantID),
				zap.Error(err))
			i18n.RespondWithError(c, i18n.ErrorTenantNotFound)
			return
		}

		name := tenant.Name
		for _, server := range servers {
			if server.Tenant == name {
				filteredServers = append(filteredServers, server)
			}
		}
		h.logger.Debug("filtered servers by tenant name",
			zap.String("tenant_name", name),
			zap.Int("server_count", len(filteredServers)))
	} else if user.Role != database.RoleAdmin {
		userTenants, err := h.db.GetUserTenants(c.Request.Context(), user.ID)
		if err != nil {
			h.logger.Error("failed to get user tenants",
				zap.Uint("user_id", user.ID),
				zap.Error(err))
			i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get user tenants: "+err.Error()))
			return
		}

		tenantNames := make([]string, len(userTenants))
		for i, tenant := range userTenants {
			tenantNames[i] = tenant.Name
		}

		for _, server := range servers {
			for _, name := range tenantNames {
				if server.Tenant == name {
					filteredServers = append(filteredServers, server)
					break
				}
			}
		}
		h.logger.Debug("filtered servers by user tenants",
			zap.Uint("user_id", user.ID),
			zap.Int("tenant_count", len(tenantNames)),
			zap.Int("server_count", len(filteredServers)))
	} else {
		filteredServers = servers
		h.logger.Debug("admin user, returning all servers",
			zap.Int("server_count", len(servers)))
	}

	// TODO: temporary
	results := make([]*dto.MCPServer, len(filteredServers))
	for i, server := range filteredServers {
		s, _ := yaml.Marshal(server)
		results[i] = &dto.MCPServer{
			Name:   server.Name,
			Config: string(s),
		}
	}

	h.logger.Info("returning MCP server list",
		zap.Int("server_count", len(results)))
	c.JSON(http.StatusOK, results)
}

func (h *MCP) HandleMCPServerCreate(c *gin.Context) {
	h.logger.Info("handling MCP server create request")

	// Read the raw YAML content from request body
	content, err := c.GetRawData()
	if err != nil {
		h.logger.Error("failed to read request body", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Failed to read request body: "+err.Error()))
		return
	}

	// Validate the YAML content and get the server name
	var cfg config.MCPConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		h.logger.Error("invalid YAML content", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Invalid YAML content: "+err.Error()))
		return
	}

	if cfg.Name == "" {
		h.logger.Warn("server name is required in configuration")
		i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Server name is required in configuration"))
		return
	}

	h.logger.Debug("validating server configuration",
		zap.String("server_name", cfg.Name),
		zap.String("tenant", cfg.Tenant))

	_, err = h.checkTenantPermission(c, cfg.Tenant, &cfg)
	if err != nil {
		h.logger.Warn("tenant permission check failed",
			zap.String("tenant", cfg.Tenant),
			zap.Error(err))
		if errors.Is(err, i18n.ErrUnauthorized) {
			i18n.RespondWithError(c, i18n.ErrUnauthorized)
		} else if errors.Is(err, i18n.ErrorTenantPermissionError) {
			i18n.RespondWithError(c, i18n.ErrorTenantPermissionError)
		} else {
			i18n.RespondWithError(c, err)
		}
		return
	}

	// Check if server already exists
	_, err = h.store.Get(c.Request.Context(), cfg.Name)
	if err == nil {
		h.logger.Warn("MCP server already exists",
			zap.String("server_name", cfg.Name))
		i18n.RespondWithError(c, i18n.ErrorMCPServerExists.WithParam("Name", cfg.Name))
		return
	}

	// Get all existing configurations
	configs, err := h.store.List(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to get existing configurations", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get existing configurations: "+err.Error()))
		return
	}

	// Add the new configuration to the list
	configs = append(configs, &cfg)

	// Validate all configurations
	if err := config.ValidateMCPConfigs(configs); err != nil {
		var validationErr *config.ValidationError
		if errors.As(err, &validationErr) {
			h.logger.Error("configuration validation failed",
				zap.String("validation_error", validationErr.Error()))
			i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Configuration validation failed: "+validationErr.Error()))
		} else {
			h.logger.Error("failed to validate configurations", zap.Error(err))
			i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Failed to validate configurations: "+err.Error()))
		}
		return
	}

	if err := h.store.Create(c.Request.Context(), &cfg); err != nil {
		h.logger.Error("failed to create MCP server",
			zap.String("server_name", cfg.Name),
			zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to create MCP server: "+err.Error()))
		return
	}

	// Send reload signal to gateway using notifier
	if err := h.notifier.NotifyUpdate(c.Request.Context(), &cfg); err != nil {
		h.logger.Error("failed to reload gateway", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to reload gateway: "+err.Error()))
		return
	}

	h.logger.Info("MCP server created successfully",
		zap.String("server_name", cfg.Name))
	i18n.Created(i18n.SuccessMCPServerCreated).With("status", "success").Send(c)
}

func (h *MCP) HandleMCPServerDelete(c *gin.Context) {
	// Get the server name from path parameter
	name := c.Param("name")
	if name == "" {
		h.logger.Warn("MCP server name required but missing")
		i18n.RespondWithError(c, i18n.ErrorMCPServerNameRequired)
		return
	}

	h.logger.Info("handling MCP server delete request",
		zap.String("server_name", name))

	// Check if server exists
	_, err := h.store.Get(c.Request.Context(), name)
	if err != nil {
		h.logger.Error("MCP server not found",
			zap.String("server_name", name),
			zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrorMCPServerNotFound.WithParam("Name", name))
		return
	}

	// Delete server
	if err := h.store.Delete(c.Request.Context(), name); err != nil {
		h.logger.Error("failed to delete MCP server",
			zap.String("server_name", name),
			zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to delete MCP server: "+err.Error()))
		return
	}

	h.logger.Info("MCP server deleted successfully",
		zap.String("server_name", name))
	i18n.Success(i18n.SuccessMCPServerDeleted).With("status", "success").Send(c)
}

func (h *MCP) HandleMCPServerSync(c *gin.Context) {
	h.logger.Info("handling MCP server sync request")

	// Send reload signal to gateway using notifier
	if err := h.notifier.NotifyUpdate(c.Request.Context(), nil); err != nil {
		h.logger.Error("failed to reload gateway", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to reload gateway: "+err.Error()))
		return
	}

	h.logger.Info("MCP servers synced successfully")
	i18n.Success(i18n.SuccessMCPServerSynced).With("status", "success").Send(c)
}

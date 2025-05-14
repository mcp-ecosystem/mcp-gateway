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
	"gopkg.in/yaml.v3"
)

type MCP struct {
	db       database.Database
	store    storage.Store
	notifier notifier.Notifier
}

func NewMCP(db database.Database, store storage.Store, ntf notifier.Notifier) *MCP {
	return &MCP{
		db:       db,
		store:    store,
		notifier: ntf,
	}
}

// checkTenantPermission checks if the user has permission to access the specified tenant and
// verifies that all router prefixes start with the tenant prefix as a complete path segment
func (h *MCP) checkTenantPermission(c *gin.Context, tenantName string, cfg *config.MCPConfig) (*database.Tenant, error) {
	// Check if tenant name is empty
	if tenantName == "" {
		return nil, i18n.ErrorTenantNameRequired
	}

	// Get user authentication information
	claims, exists := c.Get("claims")
	if !exists {
		return nil, i18n.ErrUnauthorized
	}
	jwtClaims := claims.(*jwt.Claims)

	// Get user information
	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		return nil, i18n.ErrInternalServer.WithParam("Reason", "Failed to get user info: "+err.Error())
	}

	// Get tenant information
	tenant, err := h.db.GetTenantByName(c.Request.Context(), tenantName)
	if err != nil {
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
			return nil, i18n.ErrorRouterPrefixError
		}
	}

	// Check user permission if not admin
	if user.Role != database.RoleAdmin {
		userTenants, err := h.db.GetUserTenants(c.Request.Context(), user.ID)
		if err != nil {
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
			return nil, i18n.ErrorTenantPermissionError
		}
	}

	return tenant, nil
}

func (h *MCP) HandleMCPServerUpdate(c *gin.Context) {
	// Get the server name from path parameter instead of query parameter
	name := c.Param("name")
	if name == "" {
		i18n.RespondWithError(c, i18n.ErrorMCPServerNameRequired)
		return
	}

	// Read the raw YAML content from request body
	content, err := c.GetRawData()
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Failed to read request body: "+err.Error()))
		return
	}

	// Validate the YAML content
	var cfg config.MCPConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Invalid YAML content: "+err.Error()))
		return
	}

	// Get existing server
	oldCfg, err := h.store.Get(c.Request.Context(), name)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrorMCPServerNotFound.WithParam("Name", name))
		return
	}

	if oldCfg.Name != cfg.Name {
		i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Server name in configuration must match name parameter"))
		return
	}

	_, err = h.checkTenantPermission(c, cfg.Tenant, &cfg)
	if err != nil {
		i18n.RespondWithError(c, err)
		return
	}

	// Get all existing configurations
	configs, err := h.store.List(c.Request.Context())
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get existing configurations: "+err.Error()))
		return
	}

	// Replace the old configuration with the new one
	for i, c := range configs {
		if c.Name == name {
			configs[i] = &cfg
			break
		}
	}

	// Validate all configurations
	if err := config.ValidateMCPConfigs(configs); err != nil {
		var validationErr *config.ValidationError
		if errors.As(err, &validationErr) {
			i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Configuration validation failed: "+validationErr.Error()))
		} else {
			i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Failed to validate configurations: "+err.Error()))
		}
		return
	}

	if err := h.store.Update(c.Request.Context(), &cfg); err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to update MCP server: "+err.Error()))
		return
	}

	// Send reload signal to gateway using notifier
	if err := h.notifier.NotifyUpdate(c.Request.Context(), &cfg); err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to reload gateway: "+err.Error()))
		return
	}

	i18n.Success(i18n.SuccessMCPServerUpdated).With("status", "success").Send(c)
}

func (h *MCP) HandleListMCPServers(c *gin.Context) {
	tenantIDStr := c.Query("tenantId")
	var tenantID uint
	if tenantIDStr != "" {
		tid, err := strconv.ParseUint(tenantIDStr, 10, 32)
		if err != nil {
			i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Invalid tenantId parameter"))
			return
		}
		tenantID = uint(tid)
	}

	claims, exists := c.Get("claims")
	if !exists {
		i18n.RespondWithError(c, i18n.ErrUnauthorized)
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get user info: "+err.Error()))
		return
	}

	servers, err := h.store.List(c.Request.Context())
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get MCP servers: "+err.Error()))
		return
	}

	if user.Role != database.RoleAdmin && tenantID > 0 {
		userTenants, err := h.db.GetUserTenants(c.Request.Context(), user.ID)
		if err != nil {
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
			i18n.RespondWithError(c, i18n.ErrorTenantPermissionError)
			return
		}
	}

	var filteredServers []*config.MCPConfig
	if tenantID > 0 {
		tenant, err := h.db.GetTenantByID(c.Request.Context(), tenantID)
		if err != nil {
			i18n.RespondWithError(c, i18n.ErrorTenantNotFound)
			return
		}

		name := tenant.Name
		for _, server := range servers {
			if server.Tenant == name {
				filteredServers = append(filteredServers, server)
			}
		}
	} else if user.Role != database.RoleAdmin {
		userTenants, err := h.db.GetUserTenants(c.Request.Context(), user.ID)
		if err != nil {
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
	} else {
		filteredServers = servers
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

	c.JSON(http.StatusOK, results)
}

func (h *MCP) HandleMCPServerCreate(c *gin.Context) {
	// Read the raw YAML content from request body
	content, err := c.GetRawData()
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Failed to read request body: "+err.Error()))
		return
	}

	// Validate the YAML content and get the server name
	var cfg config.MCPConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Invalid YAML content: "+err.Error()))
		return
	}

	if cfg.Name == "" {
		i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Server name is required in configuration"))
		return
	}

	_, err = h.checkTenantPermission(c, cfg.Tenant, &cfg)
	if err != nil {
		if err == i18n.ErrUnauthorized {
			i18n.RespondWithError(c, i18n.ErrUnauthorized)
		} else if err == i18n.ErrorTenantPermissionError {
			i18n.RespondWithError(c, i18n.ErrorTenantPermissionError)
		} else {
			i18n.RespondWithError(c, err)
		}
		return
	}

	// Check if server already exists
	_, err = h.store.Get(c.Request.Context(), cfg.Name)
	if err == nil {
		i18n.RespondWithError(c, i18n.ErrorMCPServerExists.WithParam("Name", cfg.Name))
		return
	}

	// Get all existing configurations
	configs, err := h.store.List(c.Request.Context())
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get existing configurations: "+err.Error()))
		return
	}

	// Add the new configuration to the list
	configs = append(configs, &cfg)

	// Validate all configurations
	if err := config.ValidateMCPConfigs(configs); err != nil {
		var validationErr *config.ValidationError
		if errors.As(err, &validationErr) {
			i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Configuration validation failed: "+validationErr.Error()))
		} else {
			i18n.RespondWithError(c, i18n.ErrorMCPServerValidation.WithParam("Reason", "Failed to validate configurations: "+err.Error()))
		}
		return
	}

	if err := h.store.Create(c.Request.Context(), &cfg); err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to create MCP server: "+err.Error()))
		return
	}

	// Send reload signal to gateway using notifier
	if err := h.notifier.NotifyUpdate(c.Request.Context(), &cfg); err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to reload gateway: "+err.Error()))
		return
	}

	i18n.Created(i18n.SuccessMCPServerCreated).With("status", "success").Send(c)
}

func (h *MCP) HandleMCPServerDelete(c *gin.Context) {
	// Get the server name from path parameter
	name := c.Param("name")
	if name == "" {
		i18n.RespondWithError(c, i18n.ErrorMCPServerNameRequired)
		return
	}

	// Check if server exists
	_, err := h.store.Get(c.Request.Context(), name)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrorMCPServerNotFound.WithParam("Name", name))
		return
	}

	// Delete server
	if err := h.store.Delete(c.Request.Context(), name); err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to delete MCP server: "+err.Error()))
		return
	}

	i18n.Success(i18n.SuccessMCPServerDeleted).With("status", "success").Send(c)
}

func (h *MCP) HandleMCPServerSync(c *gin.Context) {
	// Send reload signal to gateway using notifier
	if err := h.notifier.NotifyUpdate(c.Request.Context(), nil); err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to reload gateway: "+err.Error()))
		return
	}

	i18n.Success(i18n.SuccessMCPServerSynced).With("status", "success").Send(c)
}

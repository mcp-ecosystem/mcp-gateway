package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/auth/jwt"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/dto"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage/notifier"
	"gopkg.in/yaml.v3"
	"net/http"
	"strconv"
	"strings"
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

func (h *MCP) HandleMCPServerUpdate(c *gin.Context) {
	// Get the server name from path parameter instead of query parameter
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name parameter is required"})
		return
	}

	// Read the raw YAML content from request body
	content, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body: " + err.Error()})
		return
	}

	// Validate the YAML content
	var cfg config.MCPConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid YAML content: " + err.Error()})
		return
	}

	// Check if the server name in config matches the name parameter
	if len(cfg.Servers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server name in configuration must match name parameter"})
		return
	}

	// Get existing server
	oldCfg, err := h.store.Get(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP server not found"})
		return
	}

	if oldCfg.Name != cfg.Name {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server name in configuration must match name parameter"})
		return
	}

	// Get all existing configurations
	configs, err := h.store.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get existing configurations: " + err.Error(),
		})
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
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "configuration validation failed: " + validationErr.Error(),
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "failed to validate configurations: " + err.Error(),
			})
		}
		return
	}

	if err := h.store.Update(c.Request.Context(), &cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update MCP server: " + err.Error(),
		})
		return
	}

	// Send reload signal to gateway using notifier
	if err := h.notifier.NotifyUpdate(c.Request.Context(), &cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to reload gateway: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func (h *MCP) HandleListMCPServers(c *gin.Context) {
	// 获取租户ID参数
	tenantIDStr := c.Query("tenantId")
	var tenantID uint
	if tenantIDStr != "" {
		tid, err := strconv.ParseUint(tenantIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid tenantId parameter",
			})
			return
		}
		tenantID = uint(tid)
	}

	// 获取认证信息
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	jwtClaims := claims.(*jwt.Claims)

	// 获取用户信息
	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user info: " + err.Error(),
		})
		return
	}

	// 获取网关服务器列表
	servers, err := h.store.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get MCP servers: " + err.Error(),
		})
		return
	}

	// 如果是普通用户且指定了租户ID，检查用户是否有该租户的权限
	if user.Role != database.RoleAdmin && tenantID > 0 {
		userTenants, err := h.db.GetUserTenants(c.Request.Context(), user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get user tenants: " + err.Error(),
			})
			return
		}

		// 检查指定租户是否在用户的授权租户列表中
		hasPermission := false
		for _, tenant := range userTenants {
			if tenant.ID == tenantID {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "User does not have permission to access this tenant",
			})
			return
		}
	}

	// 过滤结果
	var filteredServers []*config.MCPConfig
	if tenantID > 0 {
		// 如果指定了租户ID，根据租户ID过滤
		tenant, err := h.db.GetTenantByID(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Tenant not found",
			})
			return
		}

		// 根据租户前缀过滤服务器
		prefix := tenant.Prefix
		for _, server := range servers {
			// 检查server的namespace是否与租户前缀匹配
			for _, srv := range server.Servers {
				if strings.HasPrefix(srv.Namespace, prefix) {
					filteredServers = append(filteredServers, server)
					break
				}
			}
		}
	} else if user.Role != database.RoleAdmin {
		// 如果是普通用户且未指定租户ID，获取该用户有权限的所有租户
		userTenants, err := h.db.GetUserTenants(c.Request.Context(), user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get user tenants: " + err.Error(),
			})
			return
		}

		tenantPrefixes := make([]string, len(userTenants))
		for i, tenant := range userTenants {
			tenantPrefixes[i] = tenant.Prefix
		}

		// 根据用户有权限的租户前缀过滤服务器
		for _, server := range servers {
			for _, srv := range server.Servers {
				for _, prefix := range tenantPrefixes {
					if strings.HasPrefix(srv.Namespace, prefix) {
						filteredServers = append(filteredServers, server)
						break
					}
				}
			}
		}
	} else {
		// 如果是管理员且未指定租户ID，返回所有服务器
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body: " + err.Error()})
		return
	}

	// Validate the YAML content and get the server name
	var cfg config.MCPConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid YAML content: " + err.Error()})
		return
	}

	// Check if there is at least one server in the config
	if len(cfg.Servers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no server configuration found in YAML"})
		return
	}

	if cfg.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server name is required in configuration"})
		return
	}

	// Check if server already exists
	_, err = h.store.Get(c.Request.Context(), cfg.Name)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "MCP server already exists"})
		return
	}

	// Get all existing configurations
	configs, err := h.store.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get existing configurations: " + err.Error(),
		})
		return
	}

	// Add the new configuration to the list
	configs = append(configs, &cfg)

	// Validate all configurations
	if err := config.ValidateMCPConfigs(configs); err != nil {
		var validationErr *config.ValidationError
		if errors.As(err, &validationErr) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "configuration validation failed: " + validationErr.Error(),
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "failed to validate configurations: " + err.Error(),
			})
		}
		return
	}

	if err := h.store.Create(c.Request.Context(), &cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create MCP server: " + err.Error(),
		})
		return
	}

	// Send reload signal to gateway using notifier
	if err := h.notifier.NotifyUpdate(c.Request.Context(), &cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to reload gateway: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
	})
}

func (h *MCP) HandleMCPServerDelete(c *gin.Context) {
	// Get the server name from path parameter
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name parameter is required"})
		return
	}

	// Check if server exists
	_, err := h.store.Get(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP server not found"})
		return
	}

	// Delete server
	if err := h.store.Delete(c.Request.Context(), name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete MCP server: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func (h *MCP) HandleMCPServerSync(c *gin.Context) {
	// Send reload signal to gateway using notifier
	if err := h.notifier.NotifyUpdate(c.Request.Context(), nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to reload gateway: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/auth/jwt"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"
	"github.com/mcp-ecosystem/mcp-gateway/internal/i18n"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage"
	"go.uber.org/zap"
)

type Registry struct {
	db     database.Database
	store  storage.Store
	logger *zap.Logger
}

func NewRegistry(db database.Database, store storage.Store, logger *zap.Logger) *Registry {
	return &Registry{
		db:     db,
		store:  store,
		logger: logger,
	}
}

type PaginatedResponse struct {
	Servers    []ServerResponse `json:"servers"`
	Next       string           `json:"next,omitempty"`
	TotalCount int              `json:"total_count"`
}

type ServerResponse struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Repository    *config.RepositoryConfig `json:"repository,omitempty"`
	VersionDetail VersionDetail          `json:"version_detail"`
}

type VersionDetail struct {
	Version     string `json:"version"`
	ReleaseDate string `json:"release_date,omitempty"`
	IsLatest    bool   `json:"is_latest"`
}

type ServerDetailResponse struct {
	ServerResponse
	Packages []interface{} `json:"packages,omitempty"`
	Remotes  []interface{} `json:"remotes,omitempty"`
}

func (h *Registry) HandleListServers(c *gin.Context) {
	h.logger.Info("handling registry servers list request")

	cursor := c.Query("cursor")
	limitStr := c.Query("limit")
	limit := 30 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 100 {
				limit = 100 // cap maximum
			}
		}
	}

	tenantName := h.getUserTenantContext(c)

	servers, nextCursor, err := h.store.ListRegistryServers(c.Request.Context(), cursor, limit, tenantName)
	if err != nil {
		h.logger.Error("failed to list registry servers", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to list servers: "+err.Error()))
		return
	}

	response := PaginatedResponse{
		Servers:    make([]ServerResponse, 0, len(servers)),
		TotalCount: len(servers),
	}

	for _, server := range servers {
		response.Servers = append(response.Servers, ServerResponse{
			ID:          server.ID,
			Name:        server.Name,
			Description: server.Description,
			Repository:  server.Repository,
			VersionDetail: VersionDetail{
				Version:  server.Version,
				IsLatest: true,
			},
		})
	}

	if nextCursor != "" {
		response.Next = nextCursor
	}

	c.JSON(http.StatusOK, response)
}

func (h *Registry) HandleGetServerDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Server ID required"))
		return
	}

	tenantName := h.getUserTenantContext(c)
	server, err := h.store.GetRegistryServerByID(c.Request.Context(), id, tenantName)
	if err != nil {
		h.logger.Error("failed to get registry server", zap.String("id", id), zap.Error(err))
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		} else {
			i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", err.Error()))
		}
		return
	}

	response := ServerDetailResponse{
		ServerResponse: ServerResponse{
			ID:          server.ID,
			Name:        server.Name,
			Description: server.Description,
			Repository:  server.Repository,
			VersionDetail: VersionDetail{
				Version:  server.Version,
				IsLatest: true,
			},
		},
		Packages: []interface{}{}, // TODO: extract from MCPConfig
		Remotes:  []interface{}{}, // TODO: extract from MCPConfig
	}

	c.JSON(http.StatusOK, response)
}

func (h *Registry) HandlePublishServer(c *gin.Context) {
	var request struct {
		Name        string                   `json:"name" binding:"required"`
		Description string                   `json:"description"`
		Repository  *config.RepositoryConfig `json:"repository"`
		Version     string                   `json:"version" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", err.Error()))
		return
	}

	server, err := h.store.Get(c.Request.Context(), request.Name)
	if err != nil {
		h.logger.Error("server not found for publishing", zap.String("name", request.Name), zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrorMCPServerNotFound.WithParam("Name", request.Name))
		return
	}

	tenantName := h.getUserTenantContext(c)
	if server.Tenant != tenantName {
		i18n.RespondWithError(c, i18n.ErrorTenantPermissionError)
		return
	}

	server.Description = request.Description
	server.Repository = request.Repository
	server.Version = request.Version

	if err := h.store.PublishToRegistry(c.Request.Context(), server); err != nil {
		h.logger.Error("failed to publish server", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Server publication successful",
		"id":      server.ID,
	})
}

func (h *Registry) getUserTenantContext(c *gin.Context) string {
	claims, exists := c.Get("claims")
	if !exists {
		return ""
	}
	jwtClaims := claims.(*jwt.Claims)

	user, err := h.db.GetUserByUsername(c.Request.Context(), jwtClaims.Username)
	if err != nil || user.Role == database.RoleAdmin {
		return "" // Admin can see all
	}

	tenants, err := h.db.GetUserTenants(c.Request.Context(), user.ID)
	if err != nil || len(tenants) == 0 {
		return ""
	}
	return tenants[0].Name
}

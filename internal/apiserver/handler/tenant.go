package handler

import (
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/i18n"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/dto"
	"go.uber.org/zap"
)

// ListTenants handles listing all tenants
func (h *Handler) ListTenants(c *gin.Context) {
	h.logger.Info("listing all tenants",
		zap.String("remote_addr", c.ClientIP()))

	tenants, err := h.db.ListTenants(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to list tenants",
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	h.logger.Debug("successfully retrieved tenants list",
		zap.Int("tenant_count", len(tenants)),
		zap.String("remote_addr", c.ClientIP()))

	i18n.Success(i18n.SuccessTenantInfo).WithPayload(tenants).Send(c)
}

// CreateTenant handles tenant creation
func (h *Handler) CreateTenant(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid create tenant request",
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.Error(i18n.ErrBadRequest.WithParam("Reason", err.Error())).Send(c)
		return
	}

	h.logger.Info("processing tenant creation request",
		zap.String("tenant_name", req.Name),
		zap.String("tenant_prefix", req.Prefix),
		zap.String("remote_addr", c.ClientIP()))

	if req.Name == "" || req.Prefix == "" {
		h.logger.Warn("tenant creation missing required fields",
			zap.String("name", req.Name),
			zap.String("prefix", req.Prefix),
			zap.String("remote_addr", c.ClientIP()))
		i18n.From(i18n.ErrorTenantRequiredFields).Send(c)
		return
	}

	existingTenants, err := h.db.ListTenants(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to list existing tenants for validation",
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	for _, tenant := range existingTenants {
		if tenant.Name == req.Name {
			h.logger.Warn("tenant with same name already exists",
				zap.String("tenant_name", req.Name),
				zap.String("remote_addr", c.ClientIP()))
			i18n.From(i18n.ErrorTenantNameExists).Send(c)
			return
		}
		if tenant.Prefix == req.Prefix {
			h.logger.Warn("tenant with same prefix already exists",
				zap.String("tenant_prefix", req.Prefix),
				zap.String("remote_addr", c.ClientIP()))
			i18n.From(i18n.ErrorTenantPrefixExists).Send(c)
			return
		}
	}

	newTenant := &database.Tenant{
		Name:        req.Name,
		Prefix:      req.Prefix,
		Description: req.Description,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.db.CreateTenant(c.Request.Context(), newTenant); err != nil {
		h.logger.Error("failed to create tenant in database",
			zap.Error(err),
			zap.String("tenant_name", req.Name),
			zap.String("tenant_prefix", req.Prefix),
			zap.String("remote_addr", c.ClientIP()))
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	h.logger.Info("tenant created successfully",
		zap.String("tenant_name", newTenant.Name),
		zap.String("tenant_prefix", newTenant.Prefix),
		zap.Uint("tenant_id", newTenant.ID),
		zap.String("remote_addr", c.ClientIP()))

	i18n.Created(i18n.SuccessTenantCreated).With("id", newTenant.ID).Send(c)
}

// UpdateTenant handles tenant updates
func (h *Handler) UpdateTenant(c *gin.Context) {
	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid update tenant request",
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.BadRequest("ErrorInvalidRequest").WithParam("Reason", err.Error()).Send(c)
		return
	}

	h.logger.Info("processing tenant update request",
		zap.String("tenant_name", req.Name),
		zap.String("remote_addr", c.ClientIP()))

	existingTenant, err := h.db.GetTenantByName(c.Request.Context(), req.Name)
	if err != nil {
		h.logger.Warn("tenant not found for update",
			zap.Error(err),
			zap.String("tenant_name", req.Name),
			zap.String("remote_addr", c.ClientIP()))
		i18n.NotFoundFromErr(i18n.ErrorTenantNotFound.WithParam("Name", req.Name)).Send(c)
		return
	}

	h.logger.Debug("found existing tenant for update",
		zap.String("tenant_name", existingTenant.Name),
		zap.String("tenant_prefix", existingTenant.Prefix),
		zap.Uint("tenant_id", existingTenant.ID),
		zap.String("remote_addr", c.ClientIP()))

	allTenants, err := h.db.ListTenants(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to list tenants for validation",
			zap.Error(err),
			zap.String("tenant_name", req.Name),
			zap.String("remote_addr", c.ClientIP()))
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	if req.Prefix != "" && req.Prefix != existingTenant.Prefix {
		h.logger.Debug("prefix change requested",
			zap.String("old_prefix", existingTenant.Prefix),
			zap.String("new_prefix", req.Prefix),
			zap.String("tenant_name", req.Name),
			zap.String("remote_addr", c.ClientIP()))

		for _, tenant := range allTenants {
			if tenant.ID != existingTenant.ID && tenant.Prefix == req.Prefix {
				h.logger.Warn("tenant prefix already exists",
					zap.String("prefix", req.Prefix),
					zap.String("tenant_name", req.Name),
					zap.String("remote_addr", c.ClientIP()))
				i18n.From(i18n.ErrorTenantPrefixExists).Send(c)
				return
			}
		}

		existingTenant.Prefix = req.Prefix
	}

	if req.Description != "" {
		existingTenant.Description = req.Description
	}
	if req.IsActive != nil {
		existingTenant.IsActive = *req.IsActive
	}
	existingTenant.UpdatedAt = time.Now()

	if err := h.db.UpdateTenant(c.Request.Context(), existingTenant); err != nil {
		h.logger.Error("failed to update tenant in database",
			zap.Error(err),
			zap.String("tenant_name", req.Name),
			zap.Uint("tenant_id", existingTenant.ID),
			zap.String("remote_addr", c.ClientIP()))
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	h.logger.Info("tenant updated successfully",
		zap.String("tenant_name", existingTenant.Name),
		zap.String("tenant_prefix", existingTenant.Prefix),
		zap.Uint("tenant_id", existingTenant.ID),
		zap.String("remote_addr", c.ClientIP()))

	i18n.Success(i18n.SuccessTenantUpdated).Send(c)
}

// DeleteTenant handles tenant deletion
func (h *Handler) DeleteTenant(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		h.logger.Warn("missing tenant name parameter",
			zap.String("remote_addr", c.ClientIP()))
		i18n.Error(i18n.ErrorTenantNameRequired).Send(c)
		return
	}

	h.logger.Info("processing tenant deletion request",
		zap.String("tenant_name", name),
		zap.String("remote_addr", c.ClientIP()))

	existingTenant, err := h.db.GetTenantByName(c.Request.Context(), name)
	if err != nil {
		h.logger.Warn("tenant not found for deletion",
			zap.Error(err),
			zap.String("tenant_name", name),
			zap.String("remote_addr", c.ClientIP()))
		i18n.NotFoundFromErr(i18n.ErrorTenantNotFound.WithParam("Name", name)).Send(c)
		return
	}

	if err := h.db.DeleteTenant(c.Request.Context(), existingTenant.ID); err != nil {
		h.logger.Error("failed to delete tenant from database",
			zap.Error(err),
			zap.String("tenant_name", name),
			zap.Uint("tenant_id", existingTenant.ID),
			zap.String("remote_addr", c.ClientIP()))
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	h.logger.Info("tenant deleted successfully",
		zap.String("tenant_name", name),
		zap.Uint("tenant_id", existingTenant.ID),
		zap.String("remote_addr", c.ClientIP()))

	i18n.Success(i18n.SuccessTenantDeleted).Send(c)
}

// GetTenantInfo handles getting tenant info by name
func (h *Handler) GetTenantInfo(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		h.logger.Warn("missing tenant name parameter",
			zap.String("remote_addr", c.ClientIP()))
		i18n.Error(i18n.ErrorTenantNameRequired).Send(c)
		return
	}

	h.logger.Info("retrieving tenant information",
		zap.String("tenant_name", name),
		zap.String("remote_addr", c.ClientIP()))

	tenant, err := h.db.GetTenantByName(c.Request.Context(), name)
	if err != nil {
		h.logger.Warn("tenant not found",
			zap.Error(err),
			zap.String("tenant_name", name),
			zap.String("remote_addr", c.ClientIP()))
		i18n.NotFoundFromErr(i18n.ErrorTenantNotFound.WithParam("Name", name)).Send(c)
		return
	}

	h.logger.Debug("successfully retrieved tenant information",
		zap.String("tenant_name", tenant.Name),
		zap.String("tenant_prefix", tenant.Prefix),
		zap.Uint("tenant_id", tenant.ID),
		zap.String("remote_addr", c.ClientIP()))

	i18n.Success(i18n.SuccessTenantInfo).WithPayload(tenant).Send(c)
}

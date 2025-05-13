package handler

import (
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/i18n"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/dto"
)

// ListTenants handles listing all tenants
func (h *Handler) ListTenants(c *gin.Context) {
	tenants, err := h.db.ListTenants(c.Request.Context())
	if err != nil {
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	i18n.Success(i18n.SuccessTenantInfo).WithPayload(tenants).Send(c)
}

// CreateTenant handles tenant creation
func (h *Handler) CreateTenant(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		i18n.Error(i18n.ErrBadRequest.WithParam("Reason", err.Error())).Send(c)
		return
	}

	// Validate request
	if req.Name == "" || req.Prefix == "" {
		i18n.From(i18n.ErrorTenantRequiredFields).Send(c)
		return
	}

	// Check if name or prefix already exists
	existingTenants, err := h.db.ListTenants(c.Request.Context())
	if err != nil {
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	for _, tenant := range existingTenants {
		if tenant.Name == req.Name {
			i18n.From(i18n.ErrorTenantNameExists).Send(c)
			return
		}
		if tenant.Prefix == req.Prefix {
			i18n.From(i18n.ErrorTenantPrefixExists).Send(c)
			return
		}
	}

	// Create tenant
	newTenant := &database.Tenant{
		Name:        req.Name,
		Prefix:      req.Prefix,
		Description: req.Description,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.db.CreateTenant(c.Request.Context(), newTenant); err != nil {
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	i18n.Created(i18n.SuccessTenantCreated).With("id", newTenant.ID).Send(c)
}

// UpdateTenant handles tenant updates
func (h *Handler) UpdateTenant(c *gin.Context) {
	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		i18n.BadRequest("ErrorInvalidRequest").WithParam("Reason", err.Error()).Send(c)
		return
	}

	// Get the tenant from the database
	existingTenant, err := h.db.GetTenantByName(c.Request.Context(), req.Name)
	if err != nil {
		i18n.NotFoundFromErr(i18n.ErrorTenantNotFound.WithParam("Name", req.Name)).Send(c)
		return
	}

	// Get all tenants for validation
	allTenants, err := h.db.ListTenants(c.Request.Context())
	if err != nil {
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	// Check if prefix is being updated and if it already exists
	if req.Prefix != "" && req.Prefix != existingTenant.Prefix {
		for _, tenant := range allTenants {
			if tenant.ID != existingTenant.ID && tenant.Prefix == req.Prefix {
				i18n.From(i18n.ErrorTenantPrefixExists).Send(c)
				return
			}
		}

		existingTenant.Prefix = req.Prefix
	}

	// Update tenant fields
	if req.Description != "" {
		existingTenant.Description = req.Description
	}
	if req.IsActive != nil {
		existingTenant.IsActive = *req.IsActive
	}
	existingTenant.UpdatedAt = time.Now()

	if err := h.db.UpdateTenant(c.Request.Context(), existingTenant); err != nil {
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	i18n.Success(i18n.SuccessTenantUpdated).Send(c)
}

// DeleteTenant handles tenant deletion
func (h *Handler) DeleteTenant(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		i18n.Error(i18n.ErrorTenantNameRequired).Send(c)
		return
	}

	// Get the tenant from the database
	existingTenant, err := h.db.GetTenantByName(c.Request.Context(), name)
	if err != nil {
		i18n.NotFoundFromErr(i18n.ErrorTenantNotFound.WithParam("Name", name)).Send(c)
		return
	}

	// Delete the tenant
	if err := h.db.DeleteTenant(c.Request.Context(), existingTenant.ID); err != nil {
		i18n.From(i18n.ErrInternalServer).Send(c)
		return
	}

	i18n.Success(i18n.SuccessTenantDeleted).Send(c)
}

// GetTenantInfo handles getting tenant info by name
func (h *Handler) GetTenantInfo(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		i18n.Error(i18n.ErrorTenantNameRequired).Send(c)
		return
	}

	// Get the tenant from the database
	tenant, err := h.db.GetTenantByName(c.Request.Context(), name)
	if err != nil {
		i18n.NotFoundFromErr(i18n.ErrorTenantNotFound.WithParam("Name", name)).Send(c)
		return
	}

	i18n.Success(i18n.SuccessTenantInfo).WithPayload(tenant).Send(c)
}

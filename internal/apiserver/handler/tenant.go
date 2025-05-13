package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/dto"
)

// ListTenants handles listing all tenants
func (h *Handler) ListTenants(c *gin.Context) {
	tenants, err := h.db.ListTenants(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenants)
}

// CreateTenant handles tenant creation
func (h *Handler) CreateTenant(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate request
	if req.Name == "" || req.Prefix == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant name and prefix are required"})
		return
	}

	// Check if name or prefix already exists
	existingTenants, err := h.db.ListTenants(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, tenant := range existingTenants {
		if tenant.Name == req.Name {
			c.JSON(http.StatusConflict, gin.H{"error": "Tenant name already exists"})
			return
		}
		if tenant.Prefix == req.Prefix {
			c.JSON(http.StatusConflict, gin.H{"error": "Prefix already exists"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": newTenant.ID})
}

// UpdateTenant handles tenant updates
func (h *Handler) UpdateTenant(c *gin.Context) {
	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the tenant from the database
	existingTenant, err := h.db.GetTenantByName(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		return
	}

	// Get all tenants for validation
	allTenants, err := h.db.ListTenants(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if prefix is being updated and if it already exists
	if req.Prefix != "" && req.Prefix != existingTenant.Prefix {
		for _, tenant := range allTenants {
			if tenant.ID != existingTenant.ID && tenant.Prefix == req.Prefix {
				c.JSON(http.StatusConflict, gin.H{"error": "Prefix already exists"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenant updated successfully"})
}

// DeleteTenant handles tenant deletion
func (h *Handler) DeleteTenant(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant name is required"})
		return
	}

	// Get the tenant from the database
	existingTenant, err := h.db.GetTenantByName(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		return
	}

	// Delete the tenant
	if err := h.db.DeleteTenant(c.Request.Context(), existingTenant.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenant deleted successfully"})
}

// GetTenantInfo handles getting tenant info by name
func (h *Handler) GetTenantInfo(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant name is required"})
		return
	}

	// Get the tenant from the database
	tenant, err := h.db.GetTenantByName(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

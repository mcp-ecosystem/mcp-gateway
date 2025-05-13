package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/i18n"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage/notifier"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/openapi"
)

// OpenAPI handles OpenAPI related operations
type OpenAPI struct {
	db       database.Database
	store    storage.Store
	notifier notifier.Notifier
}

// NewOpenAPI creates a new OpenAPI handler
func NewOpenAPI(db database.Database, store storage.Store, ntf notifier.Notifier) *OpenAPI {
	return &OpenAPI{
		db:       db,
		store:    store,
		notifier: ntf,
	}
}

// HandleImport handles OpenAPI import requests
func (h *OpenAPI) HandleImport(c *gin.Context) {
	// Get the file from the request
	file, err := c.FormFile("file")
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Failed to get file: "+err.Error()))
		return
	}

	// Open the file
	f, err := file.Open()
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to open file: "+err.Error()))
		return
	}
	defer f.Close()

	// Read the file content
	content := make([]byte, file.Size)
	if _, err := f.Read(content); err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to read file: "+err.Error()))
		return
	}

	// Create converter
	converter := openapi.NewConverter()

	// Convert the OpenAPI specification
	config, err := converter.Convert(content)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Failed to convert OpenAPI specification: "+err.Error()))
		return
	}

	// Create the MCP server configuration
	if err := h.store.Create(c.Request.Context(), config); err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to create MCP server: "+err.Error()))
		return
	}

	// Notify the gateway about the update
	if err := h.notifier.NotifyUpdate(c.Request.Context(), config); err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to notify gateway: "+err.Error()))
		return
	}

	i18n.Created(i18n.SuccessOpenAPIImported).
		With("status", "success").
		With("config", config).
		Send(c)
}

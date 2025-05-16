package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/i18n"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/storage/notifier"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/swagger"
	"go.uber.org/zap"
)

// Swagger handles Swagger 2.0 related operations
type Swagger struct {
	db       database.Database
	store    storage.Store
	notifier notifier.Notifier
	logger   *zap.Logger
}

// NewSwagger creates a new Swagger handler
func NewSwagger(db database.Database, store storage.Store, ntf notifier.Notifier, logger *zap.Logger) *Swagger {
	return &Swagger{
		db:       db,
		store:    store,
		notifier: ntf,
		logger:   logger,
	}
}

// HandleImport handles Swagger 2.0 import requests
func (h *Swagger) HandleImport(c *gin.Context) {
	h.logger.Info("handling Swagger import request")

	// Get the file from the request
	file, err := c.FormFile("file")
	if err != nil {
		h.logger.Error("failed to get file from request", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Failed to get file: "+err.Error()))
		return
	}

	h.logger.Debug("processing Swagger file",
		zap.String("filename", file.Filename),
		zap.Int64("size", file.Size))

	// Open the file
	f, err := file.Open()
	if err != nil {
		h.logger.Error("failed to open uploaded file",
			zap.String("filename", file.Filename),
			zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to open file: "+err.Error()))
		return
	}
	defer f.Close()

	// Read the file content
	content := make([]byte, file.Size)
	if _, err := f.Read(content); err != nil {
		h.logger.Error("failed to read file content",
			zap.String("filename", file.Filename),
			zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to read file: "+err.Error()))
		return
	}

	// Create converter
	h.logger.Debug("creating Swagger converter")
	converter := swagger.NewConverter()

	// Convert the Swagger specification
	h.logger.Debug("converting Swagger specification")
	config, err := converter.Convert(content)
	if err != nil {
		h.logger.Error("failed to convert Swagger specification", zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Failed to convert Swagger specification: "+err.Error()))
		return
	}

	h.logger.Info("Swagger specification converted successfully",
		zap.String("server_name", config.Name))

	// Create the MCP server configuration
	h.logger.Debug("creating MCP server configuration")
	if err := h.store.Create(c.Request.Context(), config); err != nil {
		h.logger.Error("failed to create MCP server",
			zap.String("server_name", config.Name),
			zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to create MCP server: "+err.Error()))
		return
	}

	// Notify the gateway about the update
	h.logger.Debug("notifying gateway about the update")
	if err := h.notifier.NotifyUpdate(c.Request.Context(), config); err != nil {
		h.logger.Error("failed to notify gateway",
			zap.String("server_name", config.Name),
			zap.Error(err))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to notify gateway: "+err.Error()))
		return
	}

	h.logger.Info("Swagger imported successfully",
		zap.String("server_name", config.Name))

	i18n.Created(i18n.SuccessSwaggerImported).
		With("status", "success").
		With("config", config).
		Send(c)
}

package handler

import (
	"strconv"

	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/i18n"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Chat struct {
	db     database.Database
	logger *zap.Logger
}

func NewChat(db database.Database, logger *zap.Logger) *Chat {
	return &Chat{
		db:     db,
		logger: logger.Named("apiserver.handler.chat"),
	}
}

func (h *Chat) HandleGetChatSessions(c *gin.Context) {
	h.logger.Info("retrieving chat sessions",
		zap.String("remote_addr", c.ClientIP()))

	sessions, err := h.db.GetSessions(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to get chat sessions",
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get chat sessions"))
		return
	}

	h.logger.Debug("successfully retrieved chat sessions",
		zap.Int("session_count", len(sessions)),
		zap.String("remote_addr", c.ClientIP()))

	i18n.Success(i18n.SuccessChatSessions).WithPayload(sessions).Send(c)
}

func (h *Chat) HandleGetChatMessages(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		h.logger.Warn("missing sessionId parameter",
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "SessionId is required"))
		return
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		} else if err != nil {
			h.logger.Debug("invalid page parameter",
				zap.String("page", pageStr),
				zap.Error(err),
				zap.String("remote_addr", c.ClientIP()))
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		} else if err != nil {
			h.logger.Debug("invalid pageSize parameter",
				zap.String("pageSize", pageSizeStr),
				zap.Error(err),
				zap.String("remote_addr", c.ClientIP()))
		}
	}

	h.logger.Info("retrieving chat messages",
		zap.String("session_id", sessionId),
		zap.Int("page", page),
		zap.Int("page_size", pageSize),
		zap.String("remote_addr", c.ClientIP()))

	messages, err := h.db.GetMessagesWithPagination(c.Request.Context(), sessionId, page, pageSize)
	if err != nil {
		h.logger.Error("failed to get chat messages",
			zap.Error(err),
			zap.String("session_id", sessionId),
			zap.Int("page", page),
			zap.Int("page_size", pageSize),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get messages"))
		return
	}

	h.logger.Debug("successfully retrieved chat messages",
		zap.String("session_id", sessionId),
		zap.Int("message_count", len(messages)),
		zap.Int("page", page),
		zap.Int("page_size", pageSize),
		zap.String("remote_addr", c.ClientIP()))

	i18n.Success(i18n.SuccessChatMessages).WithPayload(messages).Send(c)
}

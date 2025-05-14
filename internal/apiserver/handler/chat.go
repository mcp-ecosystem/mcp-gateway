package handler

import (
	"strconv"

	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/i18n"

	"github.com/gin-gonic/gin"
)

type Chat struct {
	db database.Database
}

func NewChat(db database.Database) *Chat {
	return &Chat{db: db}
}

func (h *Chat) HandleGetChatSessions(c *gin.Context) {
	sessions, err := h.db.GetSessions(c.Request.Context())
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get chat sessions"))
		return
	}
	i18n.Success(i18n.SuccessChatSessions).WithPayload(sessions).Send(c)
}

func (h *Chat) HandleGetChatMessages(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "SessionId is required"))
		return
	}

	// Get pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Get messages with pagination
	messages, err := h.db.GetMessagesWithPagination(c.Request.Context(), sessionId, page, pageSize)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to get messages"))
		return
	}

	i18n.Success(i18n.SuccessChatMessages).WithPayload(messages).Send(c)
}

package handler

import (
	"strconv"
	"time"

	"github.com/amoylab/unla/internal/apiserver/database"
	"github.com/amoylab/unla/internal/i18n"

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

// HandleDeleteChatSession handles the deletion of a chat session
func (h *Chat) HandleDeleteChatSession(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		h.logger.Warn("missing sessionId parameter",
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "SessionId is required"))
		return
	}

	h.logger.Info("deleting chat session",
		zap.String("session_id", sessionId),
		zap.String("remote_addr", c.ClientIP()))

	err := h.db.DeleteSession(c.Request.Context(), sessionId)
	if err != nil {
		h.logger.Error("failed to delete chat session",
			zap.Error(err),
			zap.String("session_id", sessionId),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to delete session"))
		return
	}

	h.logger.Debug("successfully deleted chat session",
		zap.String("session_id", sessionId),
		zap.String("remote_addr", c.ClientIP()))

	i18n.Success(i18n.SuccessChatDeleted).Send(c)
}

// HandleUpdateChatSessionTitle handles updating the title of a chat session
func (h *Chat) HandleUpdateChatSessionTitle(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		h.logger.Warn("missing sessionId parameter",
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "SessionId is required"))
		return
	}

	var request struct {
		Title string `json:"title" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warn("invalid request body",
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Invalid request body"))
		return
	}

	h.logger.Info("updating chat session title",
		zap.String("session_id", sessionId),
		zap.String("title", request.Title),
		zap.String("remote_addr", c.ClientIP()))

	err := h.db.UpdateSessionTitle(c.Request.Context(), sessionId, request.Title)
	if err != nil {
		h.logger.Error("failed to update chat session title",
			zap.Error(err),
			zap.String("session_id", sessionId),
			zap.String("title", request.Title),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to update session title"))
		return
	}

	h.logger.Debug("successfully updated chat session title",
		zap.String("session_id", sessionId),
		zap.String("title", request.Title),
		zap.String("remote_addr", c.ClientIP()))

	i18n.Success(i18n.SuccessChatUpdated).Send(c)
}

// HandleSaveChatMessage handles saving a chat message
func (h *Chat) HandleSaveChatMessage(c *gin.Context) {
	var request struct {
		ID               string `json:"id" binding:"required"`
		SessionID        string `json:"session_id" binding:"required"`
		Content          string `json:"content"`
		ReasoningContent string `json:"reasoning_content,omitempty"`
		Sender           string `json:"sender" binding:"required,oneof=user bot"`
		Timestamp        string `json:"timestamp" binding:"required"`
		ToolCalls        string `json:"toolCalls,omitempty"`
		ToolResult       string `json:"toolResult,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warn("invalid request body",
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Invalid request body"))
		return
	}

	// Require at least one of: content, toolCalls, toolResult, or reasoningContent
	if request.Content == "" && request.ToolCalls == "" && request.ToolResult == "" && request.ReasoningContent == "" {
		h.logger.Warn("message has no content",
			zap.String("message_id", request.ID),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Message must have content, tool calls, tool result, or reasoning content"))
		return
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, request.Timestamp)
	if err != nil {
		h.logger.Warn("invalid timestamp format",
			zap.Error(err),
			zap.String("timestamp", request.Timestamp),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "Invalid timestamp format"))
		return
	}

	h.logger.Info("saving chat message",
		zap.String("session_id", request.SessionID),
		zap.String("message_id", request.ID),
		zap.String("sender", request.Sender),
		zap.String("remote_addr", c.ClientIP()))

	// Check if session exists, if not create it
	exists, err := h.db.SessionExists(c.Request.Context(), request.SessionID)
	if err != nil {
		h.logger.Error("failed to check session existence",
			zap.Error(err),
			zap.String("session_id", request.SessionID),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to check session"))
		return
	}

	if !exists {
		// Session doesn't exist, create it
		// If this is the first user message, create session with title from message content
		if request.Sender == "user" && request.Content != "" {
			title := request.Content
			runes := []rune(title)
			if len(runes) > 50 {
				title = string(runes[:50]) + "..."
			}
			err = h.db.CreateSessionWithTitle(c.Request.Context(), request.SessionID, title)
			if err != nil {
				h.logger.Error("failed to create chat session with title",
					zap.Error(err),
					zap.String("session_id", request.SessionID),
					zap.String("title", title),
					zap.String("remote_addr", c.ClientIP()))
				i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to create session"))
				return
			}
			h.logger.Debug("auto-created chat session with title from first user message",
				zap.String("session_id", request.SessionID),
				zap.String("title", title),
				zap.String("remote_addr", c.ClientIP()))
		} else {
			// Create session without title
			err = h.db.CreateSession(c.Request.Context(), request.SessionID)
			if err != nil {
				h.logger.Error("failed to create chat session",
					zap.Error(err),
					zap.String("session_id", request.SessionID),
					zap.String("remote_addr", c.ClientIP()))
				i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to create session"))
				return
			}
			h.logger.Debug("auto-created chat session",
				zap.String("session_id", request.SessionID),
				zap.String("remote_addr", c.ClientIP()))
		}
	}

	message := &database.Message{
		ID:               request.ID,
		SessionID:        request.SessionID,
		Content:          request.Content,
		ReasoningContent: request.ReasoningContent,
		Sender:           request.Sender,
		Timestamp:        timestamp,
		ToolCalls:        request.ToolCalls,
		ToolResult:       request.ToolResult,
	}

	err = h.db.SaveMessage(c.Request.Context(), message)
	if err != nil {
		h.logger.Error("failed to save chat message",
			zap.Error(err),
			zap.String("session_id", request.SessionID),
			zap.String("message_id", request.ID),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to save message"))
		return
	}

	h.logger.Debug("successfully saved chat message",
		zap.String("session_id", request.SessionID),
		zap.String("message_id", request.ID),
		zap.String("sender", request.Sender),
		zap.String("remote_addr", c.ClientIP()))

	i18n.Success(i18n.SuccessChatMessageSaved).Send(c)
}

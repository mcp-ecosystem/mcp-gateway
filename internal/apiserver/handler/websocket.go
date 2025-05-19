package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/mcp-ecosystem/mcp-gateway/internal/i18n"

	"github.com/google/uuid"
	"github.com/mcp-ecosystem/mcp-gateway/internal/apiserver/database"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/dto"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/openai"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mcp-ecosystem/mcp-gateway/internal/auth/jwt"
	openaiGo "github.com/openai/openai-go"
	"go.uber.org/zap"
)

type WebSocket struct {
	db         database.Database
	openaiCli  *openai.Client
	jwtService *jwt.Service
	logger     *zap.Logger
}

func NewWebSocket(db database.Database, openaiCli *openai.Client, jwtService *jwt.Service, logger *zap.Logger) *WebSocket {
	return &WebSocket{
		db:         db,
		openaiCli:  openaiCli,
		jwtService: jwtService,
		logger:     logger.Named("apiserver.handler.websocket"),
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Should set stricter checks in production
	},
}

func (h *WebSocket) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		h.logger.Warn("websocket connection attempt without token",
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrUnauthorized.WithParam("Reason", "Token is required"))
		return
	}

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		h.logger.Warn("invalid token for websocket connection",
			zap.Error(err),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrUnauthorized.WithParam("Reason", "Invalid token"))
		return
	}

	h.logger.Debug("token validated successfully for websocket connection",
		zap.String("username", claims.Username),
		zap.String("remote_addr", c.ClientIP()))

	sessionId := c.Query("sessionId")
	if sessionId == "" {
		h.logger.Warn("websocket connection attempt without sessionId",
			zap.String("username", claims.Username),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "SessionId is required"))
		return
	}

	lang := c.Query("lang")
	if lang == "" {
		lang = cnst.LangDefault
	}
	c.Set(cnst.XLang, lang)

	exists, err := h.db.SessionExists(c.Request.Context(), sessionId)
	if err != nil {
		h.logger.Error("failed to check if session exists",
			zap.Error(err),
			zap.String("session_id", sessionId),
			zap.String("username", claims.Username),
			zap.String("remote_addr", c.ClientIP()))
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to check session"))
		return
	}

	if !exists {
		h.logger.Info("creating new chat session",
			zap.String("session_id", sessionId),
			zap.String("username", claims.Username),
			zap.String("remote_addr", c.ClientIP()))

		if err := h.db.CreateSession(c.Request.Context(), sessionId); err != nil {
			h.logger.Error("failed to create new chat session",
				zap.Error(err),
				zap.String("session_id", sessionId),
				zap.String("username", claims.Username),
				zap.String("remote_addr", c.ClientIP()))
			i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to create session"))
			return
		}

		h.logger.Debug("new chat session created successfully",
			zap.String("session_id", sessionId),
			zap.String("username", claims.Username),
			zap.String("remote_addr", c.ClientIP()))
	} else {
		h.logger.Debug("using existing chat session",
			zap.String("session_id", sessionId),
			zap.String("username", claims.Username),
			zap.String("remote_addr", c.ClientIP()))
	}

	h.logger.Info("new websocket connection attempt",
		zap.String("session_id", sessionId),
		zap.String("username", claims.Username),
		zap.String("remote_addr", c.ClientIP()))

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade websocket connection",
			zap.Error(err),
			zap.String("session_id", sessionId),
			zap.String("username", claims.Username),
			zap.String("remote_addr", c.ClientIP()))
		return
	}
	defer conn.Close()

	h.logger.Info("websocket connection established",
		zap.String("session_id", sessionId),
		zap.String("username", claims.Username),
		zap.String("remote_addr", c.ClientIP()))

	for {
		var message dto.WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			h.logger.Warn("error reading websocket message",
				zap.Error(err),
				zap.String("session_id", sessionId),
				zap.String("username", claims.Username),
				zap.String("remote_addr", c.ClientIP()))
			break
		}

		// Log received message
		h.logger.Debug("websocket message received",
			zap.String("session_id", sessionId),
			zap.String("message_type", message.Type),
			zap.String("sender", message.Sender),
			zap.String("username", claims.Username),
			zap.String("remote_addr", c.ClientIP()))

		// Process message based on type
		switch message.Type {
		case dto.MsgTypeMessage:

			// Save all incoming messages to database
			msg := &database.Message{
				ID:        uuid.New().String(),
				SessionID: sessionId,
				Content:   message.Content,
				Sender:    message.Sender,
				Timestamp: time.Now(),
			}
			if err := h.db.SaveMessage(c.Request.Context(), msg); err != nil {
				h.logger.Error("failed to save chat message",
					zap.Error(err),
					zap.String("session_id", sessionId),
					zap.String("message_id", msg.ID),
					zap.String("username", claims.Username),
					zap.String("remote_addr", c.ClientIP()))
			} else {
				h.logger.Debug("chat message saved successfully",
					zap.String("session_id", sessionId),
					zap.String("message_id", msg.ID),
					zap.String("username", claims.Username))
			}

			// Get conversation history from database
			messages, err := h.db.GetMessages(c.Request.Context(), sessionId)
			if err != nil {
				h.logger.Error("failed to get conversation history",
					zap.Error(err),
					zap.String("session_id", sessionId),
					zap.String("username", claims.Username),
					zap.String("remote_addr", c.ClientIP()))
				continue
			} else if len(messages) == 1 {
				// Extract title from the first message (first 20 UTF-8 characters)
				title := message.Content
				runes := []rune(title)
				if len(runes) > 20 {
					title = string(runes[:20])
				}

				h.logger.Debug("updating session title for new conversation",
					zap.String("session_id", sessionId),
					zap.String("title", title),
					zap.String("username", claims.Username))

				if err := h.db.UpdateSessionTitle(c.Request.Context(), sessionId, title); err != nil {
					h.logger.Error("failed to update session title",
						zap.Error(err),
						zap.String("session_id", sessionId),
						zap.String("title", title),
						zap.String("username", claims.Username),
						zap.String("remote_addr", c.ClientIP()))
				}
			}

			// Convert messages to OpenAI format
			openaiMessages := make([]openaiGo.ChatCompletionMessageParamUnion, len(messages))
			for i, msg := range messages {
				if msg.Sender == "bot" {
					if msg.ToolCalls != "" {
						// For bot messages with tool calls, use OfAssistant type with ToolCalls field
						var toolCalls []dto.ToolCall
						if err := json.Unmarshal([]byte(msg.ToolCalls), &toolCalls); err != nil {
							log.Printf("[WS] Failed to unmarshal tool calls - SessionID: %s, Error: %v", sessionId, err)
							continue
						}
						openaiToolCalls := make([]openaiGo.ChatCompletionMessageToolCallParam, len(toolCalls))
						for j, tc := range toolCalls {
							openaiToolCalls[j] = openaiGo.ChatCompletionMessageToolCallParam{
								ID:   tc.ID,
								Type: "function",
								Function: openaiGo.ChatCompletionMessageToolCallFunctionParam{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							}
						}
						openaiMessages[i] = openaiGo.ChatCompletionMessageParamUnion{
							OfAssistant: &openaiGo.ChatCompletionAssistantMessageParam{
								ToolCalls: openaiToolCalls,
							},
						}
					} else {
						// For regular bot messages, use OfAssistant type
						openaiMessages[i] = openaiGo.ChatCompletionMessageParamUnion{
							OfAssistant: &openaiGo.ChatCompletionAssistantMessageParam{
								Content: openaiGo.ChatCompletionAssistantMessageParamContentUnion{
									OfString: openaiGo.String(msg.Content),
								},
							},
						}
					}
				} else {
					if msg.ToolResult != "" {
						// For user messages with tool results, use OfTool type
						var toolResult dto.ToolResult
						if err := json.Unmarshal([]byte(msg.ToolResult), &toolResult); err != nil {
							log.Printf("[WS] Failed to unmarshal tool result - SessionID: %s, Error: %v", sessionId, err)
							continue
						}
						openaiMessages[i] = openaiGo.ChatCompletionMessageParamUnion{
							OfTool: &openaiGo.ChatCompletionToolMessageParam{
								ToolCallID: toolResult.ToolCallID,
								Content: openaiGo.ChatCompletionToolMessageParamContentUnion{
									OfString: openaiGo.String(toolResult.Result),
								},
							},
						}
					} else {
						// For regular user messages, use OfUser type
						openaiMessages[i] = openaiGo.ChatCompletionMessageParamUnion{
							OfUser: &openaiGo.ChatCompletionUserMessageParam{
								Content: openaiGo.ChatCompletionUserMessageParamContentUnion{
									OfString: openaiGo.String(msg.Content),
								},
							},
						}
					}
				}
			}

			// Convert tools to OpenAI format if provided
			var openaiTools []openaiGo.ChatCompletionToolParam
			if len(message.Tools) > 0 {
				openaiTools = make([]openaiGo.ChatCompletionToolParam, len(message.Tools))
				for i, tool := range message.Tools {
					openaiTools[i] = openaiGo.ChatCompletionToolParam{
						Function: openaiGo.FunctionDefinitionParam{
							Name:        tool.Name,
							Description: openaiGo.String(tool.Description),
							Parameters: openaiGo.FunctionParameters{
								"type":       "object",
								"properties": tool.Parameters.Properties,
								"required":   tool.Parameters.Required,
							},
							Strict: openaiGo.Bool(true),
						},
					}
				}
			}

			// Get streaming response from OpenAI
			stream, err := h.openaiCli.ChatCompletionStream(c.Request.Context(), openaiMessages, openaiTools)
			if err != nil {
				log.Printf("[WS] Failed to get OpenAI response - SessionID: %s, Error: %v", sessionId, err)
				continue
			}

			// Initialize response content
			responseContent := ""
			var toolCall *dto.ToolCall
			var toolCallArguments string

			// Process stream chunks
			for stream.Next() {
				chunk := stream.Current()
				// Check if this is a tool call
				if chunk.Choices[0].Delta.ToolCalls != nil {
					// Initialize tool call if not exists
					if toolCall == nil {
						toolCall = &dto.ToolCall{
							ID:   chunk.Choices[0].Delta.ToolCalls[0].ID,
							Type: chunk.Choices[0].Delta.ToolCalls[0].Type,
							Function: dto.ToolFunction{
								Name:      chunk.Choices[0].Delta.ToolCalls[0].Function.Name,
								Arguments: "",
							},
						}
					}

					// Accumulate arguments
					if chunk.Choices[0].Delta.ToolCalls[0].Function.Arguments != "" {
						toolCallArguments += chunk.Choices[0].Delta.ToolCalls[0].Function.Arguments
					}
					continue
				}

				// If this is the last chunk of tool call, send the complete tool call to client
				if chunk.Choices[0].FinishReason == "tool_calls" {
					var toolCalls []dto.ToolCall
					if toolCall != nil {
						toolCall.Function.Arguments = toolCallArguments
						toolCalls = append(toolCalls, *toolCall)
					}
					response := dto.WebSocketResponse{
						Type:      dto.MsgTypeToolCall,
						Content:   "",
						Sender:    "bot",
						Timestamp: time.Now().UnixMilli(),
						ID:        uuid.New().String(),
						ToolCalls: toolCalls,
					}
					if err := conn.WriteJSON(response); err != nil {
						log.Printf("[WS] Error writing tool call message - SessionID: %s, Error: %v", sessionId, err)
						break
					}

					s, err := json.Marshal(toolCalls)
					if err != nil {
						log.Printf("[WS] Failed to marshal tool calls - SessionID: %s, Error: %v", sessionId, err)
						continue
					}
					msg := &database.Message{
						ID:        uuid.New().String(),
						SessionID: sessionId,
						Content:   "",
						Sender:    "bot",
						Timestamp: time.Now(),
						ToolCalls: string(s),
					}
					if err := h.db.SaveMessage(c.Request.Context(), msg); err != nil {
						log.Printf("[WS] Failed to save tool call message - SessionID: %s, Error: %v", sessionId, err)
					}
					continue
				}

				// Handle regular content
				if chunk.Choices[0].Delta.Content != "" {
					responseContent += chunk.Choices[0].Delta.Content
					response := dto.WebSocketResponse{
						Type:      dto.MsgTypeStream,
						Content:   chunk.Choices[0].Delta.Content,
						Sender:    "bot",
						Timestamp: time.Now().UnixMilli(),
						ID:        uuid.New().String(),
					}
					if err := conn.WriteJSON(response); err != nil {
						log.Printf("[WS] Error writing stream message - SessionID: %s, Error: %v", sessionId, err)
						break
					}
				}

				// If this is the last chunk, save the complete message
				if chunk.Choices[0].FinishReason == "stop" {
					// Save the complete message to database
					dbMessage := &database.Message{
						ID:        uuid.New().String(),
						SessionID: sessionId,
						Content:   responseContent,
						Sender:    "bot",
						Timestamp: time.Now(),
					}
					if err := h.db.SaveMessage(c.Request.Context(), dbMessage); err != nil {
						log.Printf("[WS] Failed to save bot message - SessionID: %s, Error: %v", sessionId, err)
					}
				}
			}

		case dto.MsgTypeToolResult:
			s, err := json.Marshal(message.ToolResult)
			if err != nil {
				log.Printf("[WS] Failed to marshal tool result - SessionID: %s, Error: %v", sessionId, err)
				continue
			}
			msg := &database.Message{
				ID:         uuid.New().String(),
				SessionID:  sessionId,
				Content:    "",
				Sender:     "user",
				Timestamp:  time.Now(),
				ToolResult: string(s),
			}
			if err := h.db.SaveMessage(c.Request.Context(), msg); err != nil {
				log.Printf("[WS] Failed to save tool result message - SessionID: %s, Error: %v", sessionId, err)
			}

			// Get conversation history from database
			messages, err := h.db.GetMessages(c.Request.Context(), sessionId)
			if err != nil {
				log.Printf("[WS] Failed to get conversation history - SessionID: %s, Error: %v", sessionId, err)
				continue
			}

			// Convert messages to OpenAI format
			openaiMessages := make([]openaiGo.ChatCompletionMessageParamUnion, len(messages))
			for i, msg := range messages {
				if msg.Sender == "bot" {
					if msg.ToolCalls != "" {
						// For bot messages with tool calls, use OfAssistant type with ToolCalls field
						var toolCalls []dto.ToolCall
						if err := json.Unmarshal([]byte(msg.ToolCalls), &toolCalls); err != nil {
							log.Printf("[WS] Failed to unmarshal tool calls - SessionID: %s, Error: %v", sessionId, err)
							continue
						}
						openaiToolCalls := make([]openaiGo.ChatCompletionMessageToolCallParam, len(toolCalls))
						for j, tc := range toolCalls {
							openaiToolCalls[j] = openaiGo.ChatCompletionMessageToolCallParam{
								ID:   tc.ID,
								Type: "function",
								Function: openaiGo.ChatCompletionMessageToolCallFunctionParam{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							}
						}
						openaiMessages[i] = openaiGo.ChatCompletionMessageParamUnion{
							OfAssistant: &openaiGo.ChatCompletionAssistantMessageParam{
								ToolCalls: openaiToolCalls,
							},
						}
					} else {
						// For regular bot messages, use OfAssistant type
						openaiMessages[i] = openaiGo.ChatCompletionMessageParamUnion{
							OfAssistant: &openaiGo.ChatCompletionAssistantMessageParam{
								Content: openaiGo.ChatCompletionAssistantMessageParamContentUnion{
									OfString: openaiGo.String(msg.Content),
								},
							},
						}
					}
				} else {
					if msg.ToolResult != "" {
						// For user messages with tool results, use OfTool type
						var toolResult dto.ToolResult
						if err := json.Unmarshal([]byte(msg.ToolResult), &toolResult); err != nil {
							log.Printf("[WS] Failed to unmarshal tool result - SessionID: %s, Error: %v", sessionId, err)
							continue
						}
						openaiMessages[i] = openaiGo.ChatCompletionMessageParamUnion{
							OfTool: &openaiGo.ChatCompletionToolMessageParam{
								ToolCallID: toolResult.ToolCallID,
								Content: openaiGo.ChatCompletionToolMessageParamContentUnion{
									OfString: openaiGo.String(toolResult.Result),
								},
							},
						}
					} else {
						// For regular user messages, use OfUser type
						openaiMessages[i] = openaiGo.ChatCompletionMessageParamUnion{
							OfUser: &openaiGo.ChatCompletionUserMessageParam{
								Content: openaiGo.ChatCompletionUserMessageParamContentUnion{
									OfString: openaiGo.String(msg.Content),
								},
							},
						}
					}
				}
			}

			// Get streaming response from OpenAI
			stream, err := h.openaiCli.ChatCompletionStream(c.Request.Context(), openaiMessages, nil)
			if err != nil {
				log.Printf("[WS] Failed to get OpenAI response - SessionID: %s, Error: %v", sessionId, err)
				continue
			}

			// Initialize response content
			responseContent := ""

			// Process stream chunks
			for stream.Next() {
				chunk := stream.Current()
				if chunk.Choices[0].Delta.Content != "" {
					responseContent += chunk.Choices[0].Delta.Content
					response := dto.WebSocketResponse{
						Type:      dto.MsgTypeStream,
						Content:   chunk.Choices[0].Delta.Content,
						Sender:    "bot",
						Timestamp: time.Now().UnixMilli(),
						ID:        uuid.New().String(),
					}
					if err := conn.WriteJSON(response); err != nil {
						log.Printf("[WS] Error writing stream message - SessionID: %s, Error: %v", sessionId, err)
						break
					}
				}

				// If this is the last chunk, save the complete message
				if chunk.Choices[0].FinishReason == "stop" {
					// Save the complete message to database
					dbMessage := &database.Message{
						ID:        uuid.New().String(),
						SessionID: sessionId,
						Content:   responseContent,
						Sender:    "bot",
						Timestamp: time.Now(),
					}
					if err := h.db.SaveMessage(c.Request.Context(), dbMessage); err != nil {
						log.Printf("[WS] Failed to save bot message - SessionID: %s, Error: %v", sessionId, err)
					}
				}
			}
		}
	}
}

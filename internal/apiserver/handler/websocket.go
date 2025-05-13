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
)

type WebSocket struct {
	db         database.Database
	openaiCli  *openai.Client
	jwtService *jwt.Service
}

func NewWebSocket(db database.Database, openaiCli *openai.Client, jwtService *jwt.Service) *WebSocket {
	return &WebSocket{
		db:         db,
		openaiCli:  openaiCli,
		jwtService: jwtService,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Should set stricter checks in production
	},
}

func (h *WebSocket) HandleWebSocket(c *gin.Context) {
	// Token auth from query
	token := c.Query("token")
	if token == "" {
		i18n.RespondWithError(c, i18n.ErrUnauthorized.WithParam("Reason", "Token is required"))
		return
	}
	_, err := h.jwtService.ValidateToken(token)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrUnauthorized.WithParam("Reason", "Invalid token"))
		return
	}

	// Get sessionId from query parameter
	sessionId := c.Query("sessionId")
	if sessionId == "" {
		i18n.RespondWithError(c, i18n.ErrBadRequest.WithParam("Reason", "SessionId is required"))
		return
	}

	lang := c.Query("lang")
	if lang == "" {
		lang = cnst.LangDefault
	}
	c.Set(cnst.XLang, lang)

	// Check if session exists, if not create it
	exists, err := h.db.SessionExists(c.Request.Context(), sessionId)
	if err != nil {
		i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to check session"))
		return
	}
	if !exists {
		// Create new session with the provided sessionId
		if err := h.db.CreateSession(c.Request.Context(), sessionId); err != nil {
			i18n.RespondWithError(c, i18n.ErrInternalServer.WithParam("Reason", "Failed to create session"))
			return
		}
	}

	// Log connection attempt
	log.Printf("[WS] New connection attempt - SessionID: %s, RemoteAddr: %s", sessionId, c.Request.RemoteAddr)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] Failed to upgrade connection - SessionID: %s, Error: %v", sessionId, err)
		return
	}
	defer conn.Close()

	// Log successful connection
	log.Printf("[WS] Connection established - SessionID: %s, RemoteAddr: %s", sessionId, c.Request.RemoteAddr)

	for {
		var message dto.WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("[WS] Error reading message - SessionID: %s, Error: %v", sessionId, err)
			break
		}

		// Log received message
		log.Printf("[WS] Message received - SessionID: %s, Type: %s, Config: %s",
			sessionId, message.Type, message.Content)

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
				log.Printf("[WS] Failed to save message - SessionID: %s, Error: %v", sessionId, err)
			}

			// Get conversation history from database
			messages, err := h.db.GetMessages(c.Request.Context(), sessionId)
			if err != nil {
				log.Printf("[WS] Failed to get conversation history - SessionID: %s, Error: %v", sessionId, err)
				continue
			} else if len(messages) == 1 {
				// Extract title from the first message (first 20 UTF-8 characters)
				title := message.Content
				runes := []rune(title)
				if len(runes) > 20 {
					title = string(runes[:20])
				}
				if err := h.db.UpdateSessionTitle(c.Request.Context(), sessionId, title); err != nil {
					log.Printf("[WS] Failed to update session title - SessionID: %s, Error: %v", sessionId, err)
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

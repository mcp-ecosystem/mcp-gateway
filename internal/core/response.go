package core

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/mcp/session"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"
)

// sendProtocolError sends a protocol-level error response
func (s *Server) sendProtocolError(c *gin.Context, id any, message string, statusCode int, bizCode int) {
	response := mcp.JSONRPCErrorSchema{
		JSONRPCBaseResult: mcp.JSONRPCBaseResult{
			JSONRPC: mcp.JSPNRPCVersion,
			ID:      id,
		},
		Error: mcp.JSONRPCError{
			Code:    bizCode,
			Message: message,
		},
	}
	c.JSON(statusCode, response)
}

// sendToolExecutionError sends a tool execution error response
func (s *Server) sendToolExecutionError(c *gin.Context, conn session.Connection, req mcp.JSONRPCRequest, err error, isSSE bool) {
	response := mcp.JSONRPCResponse{
		JSONRPCBaseResult: mcp.JSONRPCBaseResult{
			JSONRPC: mcp.JSPNRPCVersion,
			ID:      req.Id,
		},
		Result: mcp.CallToolResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Error: %s", err.Error()),
				},
			},
			IsError: true,
		},
	}
	s.sendResponse(c, req.Id, conn, response, isSSE)
}

// sendSuccessResponse sends a successful response
func (s *Server) sendSuccessResponse(c *gin.Context, conn session.Connection, req mcp.JSONRPCRequest, result any, isSSE bool) {
	response := mcp.JSONRPCResponse{
		JSONRPCBaseResult: mcp.JSONRPCBaseResult{
			JSONRPC: mcp.JSPNRPCVersion,
			ID:      req.Id,
		},
		Result: result,
	}
	s.sendResponse(c, req.Id, conn, response, isSSE)
}

// sendResponse handles sending the response through SSE or direct HTTP
func (s *Server) sendResponse(c *gin.Context, id any, conn session.Connection, response interface{}, isSSE bool) {
	eventData, err := json.Marshal(response)
	if err != nil {
		s.sendProtocolError(c, id, "Failed to marshal response", http.StatusInternalServerError, mcp.ErrorCodeInternalError)
		return
	}

	if isSSE {
		// For SSE connections
		err = conn.Send(c.Request.Context(), &session.Message{
			Event: "message",
			Data:  eventData,
		})
		if err != nil {
			s.sendProtocolError(c, id, fmt.Sprintf("failed to send SSE message: %v", err), http.StatusInternalServerError, mcp.ErrorCodeInternalError)
			return
		}
		c.String(http.StatusAccepted, mcp.Accepted)
	} else {
		// For direct HTTP connections
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache") // , no-transform
		c.Header("Connection", "keep-alive")
		c.Header(mcp.HeaderMcpSessionID, conn.Meta().ID)
		c.String(http.StatusOK, fmt.Sprintf("event: message\ndata: %s\n\n", eventData))
	}
}

// sendAcceptedResponse sends an accepted response
func (s *Server) sendAcceptedResponse(c *gin.Context) {
	c.String(http.StatusAccepted, mcp.Accepted)
}

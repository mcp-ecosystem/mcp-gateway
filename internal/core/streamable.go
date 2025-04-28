package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	texttemplate "text/template"

	"github.com/mcp-ecosystem/mcp-gateway/internal/common/config"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/template"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"
)

// StreamableSession represents a session for Streamable HTTP transport
type StreamableSession struct {
	writer              http.ResponseWriter
	flusher             http.Flusher
	done                chan struct{}
	eventQueue          chan string
	sessionID           string
	notificationChannel chan mcp.JSONRPCNotification
	initialized         bool
	eventStore          *InMemoryEventStore
	mu                  sync.Mutex
	server              *Server
}

// NewStreamableSession creates a new Streamable HTTP session
func NewStreamableSession(w http.ResponseWriter, sessionID string, server *Server) (*StreamableSession, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	return &StreamableSession{
		writer:              w,
		flusher:             flusher,
		done:                make(chan struct{}),
		eventQueue:          make(chan string, 100),
		sessionID:           sessionID,
		notificationChannel: make(chan mcp.JSONRPCNotification, 100),
		eventStore:          NewInMemoryEventStore(),
		server:              server,
	}, nil
}

// SessionID returns the session ID
func (s *StreamableSession) SessionID() string {
	return s.sessionID
}

// NotificationChannel returns the notification channel
func (s *StreamableSession) NotificationChannel() chan<- mcp.JSONRPCNotification {
	return s.notificationChannel
}

// Initialize marks the session as initialized
func (s *StreamableSession) Initialize() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initialized = true
}

// Initialized returns whether the session is initialized
func (s *StreamableSession) Initialized() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initialized
}

// Close closes the session
func (s *StreamableSession) Close() {
	close(s.done)
}

// SendEvent sends an event to the client
func (s *StreamableSession) SendEvent(event string) error {
	select {
	case s.eventQueue <- event:
		return nil
	case <-s.done:
		return fmt.Errorf("session closed")
	}
}

// sendErrorResponse sends an error response through the event queue
func (s *StreamableSession) sendErrorResponse(req mcp.JSONRPCRequest, errorMsg string) {
	response := mcp.JSONRPCResponse{
		JSONRPCBaseResult: mcp.JSONRPCBaseResult{
			JSONRPC: mcp.JSPNRPCVersion,
			ID:      req.Id,
		},
		Result: mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				{
					Type: "text",
					Text: errorMsg,
				},
			},
		},
	}
	eventData, err := json.Marshal(response)
	if err != nil {
		return
	}
	select {
	case s.eventQueue <- fmt.Sprintf("event: message\ndata: %s\n\n", eventData):
		// Event queued successfully
	case <-s.done:
		// Session is closed
	}
}

// HandleRequest handles an incoming request
func (s *StreamableSession) HandleRequest(c *gin.Context, req mcp.JSONRPCRequest) error {
	// Store the request in the event store
	s.eventStore.StoreRequest(req)

	// Process the request based on its method
	switch req.Method {
	case mcp.Initialize:
		// Handle initialization request
		var params mcp.InitializeRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendErrorResponse(req, fmt.Sprintf("invalid initialize parameters: %v", err))
			return nil
		}

		response := mcp.InitializeResult{
			JSONRPCBaseResult: mcp.JSONRPCBaseResult{
				JSONRPC: mcp.JSPNRPCVersion,
				ID:      req.Id,
			},
			Result: mcp.InitializedResult{
				ProtocolVersion: mcp.LatestProtocolVersion,
				Capabilities: mcp.ServerCapabilitiesSchema{
					Logging: mcp.LoggingCapabilitySchema{},
					Tools: mcp.ToolsCapabilitySchema{
						ListChanged: true,
					},
				},
				ServerInfo: mcp.ImplementationSchema{
					Name:    "mcp-gateway",
					Version: "1.0.0",
				},
			},
		}

		eventData, err := json.Marshal(response)
		if err != nil {
			s.sendErrorResponse(req, fmt.Sprintf("failed to marshal response: %v", err))
			return nil
		}

		// Send response directly
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.String(http.StatusOK, fmt.Sprintf("event: message\ndata: %s\n\n", eventData))
		return nil

	case mcp.NotificationInitialized:
		// Set response headers
		c.Header("X-Powered-By", "Express")
		c.Header("Connection", "keep-alive")
		c.Header("Keep-Alive", "timeout=5")
		c.Header("Transfer-Encoding", "chunked")

		// Return accepted response
		c.String(http.StatusAccepted, mcp.Accepted)
		return nil

	case mcp.ToolsList:
		// Get the prefix for this session
		prefixI, ok := s.server.sessionToPrefix.Load(s.sessionID)
		if !ok {
			s.sendErrorResponse(req, "Session not found")
			return nil
		}
		prefix := prefixI.(string)

		// Get tools for this prefix
		tools, ok := s.server.prefixToTools[prefix]
		if !ok {
			tools = []mcp.ToolSchema{}
		}

		response := mcp.JSONRPCResponse{
			JSONRPCBaseResult: mcp.JSONRPCBaseResult{
				JSONRPC: mcp.JSPNRPCVersion,
				ID:      req.Id,
			},
			Result: mcp.ListToolsResult{
				Tools: tools,
			},
		}

		eventData, err := json.Marshal(response)
		if err != nil {
			s.sendErrorResponse(req, fmt.Sprintf("failed to marshal response: %v", err))
			return nil
		}

		// Send response directly
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.String(http.StatusOK, fmt.Sprintf("event: message\ndata: %s\n\n", eventData))
		return nil

	case mcp.ToolsCall:
		// Handle tool call request
		var params mcp.CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendErrorResponse(req, fmt.Sprintf("invalid tool call parameters: %v", err))
			return nil
		}

		// Find the tool in the precomputed map
		tool, exists := s.server.toolMap[params.Name]
		if !exists {
			s.sendErrorResponse(req, "Tool not found")
			return nil
		}

		// Convert arguments to map[string]any
		var args map[string]any
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			s.sendErrorResponse(req, "Invalid tool arguments")
			return nil
		}

		prefixI, ok := s.server.sessionToPrefix.Load(s.sessionID)
		if !ok {
			s.sendErrorResponse(req, "Session not found")
			return nil
		}
		prefix := prefixI.(string)
		serverCfg, ok := s.server.prefixToServerConfig[prefix]

		// Execute the tool
		result, err := s.server.executeTool(tool, args, c.Request, serverCfg.Config)
		if err != nil {
			s.sendErrorResponse(req, fmt.Sprintf("Error: %s", err.Error()))
			return nil
		}

		response := mcp.JSONRPCResponse{
			JSONRPCBaseResult: mcp.JSONRPCBaseResult{
				JSONRPC: mcp.JSPNRPCVersion,
				ID:      req.Id,
			},
			Result: mcp.CallToolResult{
				Content: []mcp.Content{
					{
						Type: "text",
						Text: result,
					},
				},
			},
		}

		eventData, err := json.Marshal(response)
		if err != nil {
			s.sendErrorResponse(req, fmt.Sprintf("failed to marshal response: %v", err))
			return nil
		}

		// Send response directly
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.String(http.StatusOK, fmt.Sprintf("event: message\ndata: %s\n\n", eventData))
		return nil

	default:
		s.sendErrorResponse(req, fmt.Sprintf("unknown method: %s", req.Method))
		return nil
	}
}

// InMemoryEventStore stores events in memory
type InMemoryEventStore struct {
	mu      sync.Mutex
	events  []mcp.JSONRPCRequest
	maxSize int
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events:  make([]mcp.JSONRPCRequest, 0),
		maxSize: 1000, // Maximum number of events to store
	}
}

// StoreRequest stores a request in the event store
func (s *InMemoryEventStore) StoreRequest(req mcp.JSONRPCRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add the request to the events slice
	s.events = append(s.events, req)

	// Remove old events if we exceed the maximum size
	if len(s.events) > s.maxSize {
		s.events = s.events[1:]
	}
}

// GetEvents returns all stored events
func (s *InMemoryEventStore) GetEvents() []mcp.JSONRPCRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.events
}

// executeTool executes a tool with the given arguments
func executeTool(tool *config.ToolConfig, args json.RawMessage, req *http.Request) (string, error) {
	// Convert arguments to map[string]any
	var argMap map[string]any
	if err := json.Unmarshal(args, &argMap); err != nil {
		return "", fmt.Errorf("invalid tool arguments: %v", err)
	}

	// Execute the tool based on its method
	switch tool.Method {
	case "GET", "POST", "PUT", "DELETE", "PATCH":
		return executeHTTPTool(tool, argMap, req)
	default:
		return "", fmt.Errorf("unsupported tool method: %s", tool.Method)
	}
}

// executeHTTPTool executes an HTTP tool
func executeHTTPTool(tool *config.ToolConfig, args map[string]any, req *http.Request) (string, error) {
	// Create HTTP client
	client := &http.Client{}

	// Create template context
	tmplCtx := template.NewContext()
	tmplCtx.Args = args

	// Set request headers in template context
	for k, v := range req.Header {
		if len(v) > 0 {
			tmplCtx.Request.Headers[k] = v[0]
		}
	}

	// Parse endpoint template
	endpointTmpl, err := texttemplate.New("endpoint").Parse(tool.Endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse endpoint template: %w", err)
	}

	// Execute endpoint template
	var endpointBuf bytes.Buffer
	if err := endpointTmpl.Execute(&endpointBuf, tmplCtx); err != nil {
		return "", fmt.Errorf("failed to execute endpoint template: %w", err)
	}
	endpoint := endpointBuf.String()

	// Parse request body template if provided
	var reqBody io.Reader
	if tool.RequestBody != "" {
		bodyTmpl, err := texttemplate.New("body").Parse(tool.RequestBody)
		if err != nil {
			return "", fmt.Errorf("failed to parse request body template: %w", err)
		}

		var bodyBuf bytes.Buffer
		if err := bodyTmpl.Execute(&bodyBuf, tmplCtx); err != nil {
			return "", fmt.Errorf("failed to execute request body template: %w", err)
		}
		reqBody = &bodyBuf
	}

	// Create request
	httpReq, err := http.NewRequest(tool.Method, endpoint, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for k, v := range tool.Headers {
		headerTmpl, err := texttemplate.New("header").Parse(v)
		if err != nil {
			return "", fmt.Errorf("failed to parse header template: %w", err)
		}

		var headerBuf bytes.Buffer
		if err := headerTmpl.Execute(&headerBuf, tmplCtx); err != nil {
			return "", fmt.Errorf("failed to execute header template: %w", err)
		}
		httpReq.Header.Set(k, headerBuf.String())
	}

	// Add arguments to headers, query params, or path params
	for _, arg := range tool.Args {
		value := fmt.Sprint(args[arg.Name])
		switch strings.ToLower(arg.Position) {
		case "header":
			httpReq.Header.Set(arg.Name, value)
		case "query":
			q := httpReq.URL.Query()
			q.Add(arg.Name, value)
			httpReq.URL.RawQuery = q.Encode()
		}
	}

	// Execute request
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response body template if provided
	if tool.ResponseBody != "" {
		respTmpl, err := texttemplate.New("response").Parse(tool.ResponseBody)
		if err != nil {
			return "", fmt.Errorf("failed to parse response body template: %w", err)
		}

		// Create data map for response template
		var respData map[string]any
		if err := json.Unmarshal(respBody, &respData); err != nil {
			return "", fmt.Errorf("failed to parse response JSON: %w", err)
		}

		// Wrap response data in response.data
		tmplCtx.Response.Data = respData
		var respBuf bytes.Buffer
		if err := respTmpl.Execute(&respBuf, tmplCtx); err != nil {
			return "", fmt.Errorf("failed to execute response body template: %w", err)
		}
		return respBuf.String(), nil
	}

	return string(respBody), nil
}

// handleMCP handles MCP connections
func (s *Server) handleMCP(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodOptions:
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Status(http.StatusOK)
		return

	case http.MethodGet:
		prefix := strings.TrimSuffix(c.Request.URL.Path, "/mcp")
		if prefix == "" {
			prefix = "/"
		}

		sessionID := c.GetHeader("Mcp-Session-Id")
		if sessionID == "" {
			sessionID = uuid.New().String()
			c.Header("Mcp-Session-Id", sessionID)
		}

		session, err := NewStreamableSession(c.Writer, sessionID, s)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		s.sessionToPrefix.Store(sessionID, prefix)

		go func() {
			for {
				select {
				case event := <-session.eventQueue:
					if _, err := c.Writer.Write([]byte(event)); err != nil {
						return
					}
					c.Writer.(http.Flusher).Flush()
				case <-session.done:
					return
				}
			}
		}()

	case http.MethodPost:
		var req mcp.JSONRPCRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		sessionID := c.GetHeader("Mcp-Session-Id")
		if sessionID == "" {
			sessionID = uuid.New().String()
			c.Header("Mcp-Session-Id", sessionID)
		}

		if req.Method == mcp.Initialize {
			prefix := strings.TrimSuffix(c.Request.URL.Path, "/mcp")
			if prefix == "" {
				prefix = "/"
			}

			session, err := NewStreamableSession(c.Writer, sessionID, s)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to create session")
				return
			}

			s.sessionToPrefix.Store(sessionID, prefix)
			s.sessions.Store(sessionID, session)

			if err := session.HandleRequest(c, req); err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}
			return
		}

		sessionI, ok := s.sessions.Load(sessionID)
		if !ok {
			c.String(http.StatusBadRequest, "Invalid or expired session")
			return
		}

		session := sessionI.(*StreamableSession)
		if err := session.HandleRequest(c, req); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

	case http.MethodDelete:
		sessionID := c.GetHeader("Mcp-Session-Id")
		if sessionID == "" {
			c.String(http.StatusBadRequest, "Missing session ID")
			return
		}

		sessionI, ok := s.sessions.Load(sessionID)
		if !ok {
			c.String(http.StatusBadRequest, "Invalid or expired session")
			return
		}

		session := sessionI.(*StreamableSession)
		session.Close()
		s.sessions.Delete(sessionID)
		s.sessionToPrefix.Delete(sessionID)

		c.String(http.StatusOK, "Session terminated")

	default:
		c.String(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

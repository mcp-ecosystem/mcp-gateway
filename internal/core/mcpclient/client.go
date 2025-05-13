package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/mcp-ecosystem/mcp-gateway/internal/core/mcpclient/transport"
	"github.com/mcp-ecosystem/mcp-gateway/pkg/mcp"
)

// Client implements the MCP client.
type Client struct {
	transport transport.Interface

	initialized        bool
	notifications      []func(mcp.JSONRPCNotification)
	notifyMu           sync.RWMutex
	requestID          atomic.Int64
	clientCapabilities mcp.ClientCapabilitiesSchema
	serverCapabilities mcp.ServerCapabilitiesSchema
}

type ClientOption func(*Client)

// WithClientCapabilities sets the client capabilities for the client.
func WithClientCapabilities(capabilities mcp.ClientCapabilitiesSchema) ClientOption {
	return func(c *Client) {
		c.clientCapabilities = capabilities
	}
}

// NewClient creates a new MCP client with the given transport.
// Usage:
//
//	stdio := transport.NewStdio("./mcp_server", nil, "xxx")
//	client, err := NewClient(stdio)
//	if err != nil {
//	    log.Fatalf("Failed to create client: %v", err)
//	}
func NewClient(transport transport.Interface, options ...ClientOption) *Client {
	client := &Client{
		transport: transport,
	}

	for _, opt := range options {
		opt(client)
	}

	return client
}

// Start initiates the connection to the server.
// Must be called before using the client.
func (c *Client) Start(ctx context.Context) error {
	if c.transport == nil {
		return fmt.Errorf("transport is nil")
	}
	err := c.transport.Start(ctx)
	if err != nil {
		return err
	}

	c.transport.SetNotificationHandler(func(notification mcp.JSONRPCNotification) {
		c.notifyMu.RLock()
		defer c.notifyMu.RUnlock()
		for _, handler := range c.notifications {
			handler(notification)
		}
	})
	return nil
}

// Close shuts down the client and closes the transport.
func (c *Client) Close() error {
	return c.transport.Close()
}

// OnNotification registers a handler function to be called when notifications are received.
// Multiple handlers can be registered and will be called in the order they were added.
func (c *Client) OnNotification(
	handler func(notification mcp.JSONRPCNotification),
) {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	c.notifications = append(c.notifications, handler)
}

// sendRequest sends a JSON-RPC request to the server and waits for a response.
// Returns the raw JSON response message or an error if the request fails.
func (c *Client) sendRequest(
	ctx context.Context,
	method string,
	params json.RawMessage,
) (json.RawMessage, error) {
	if !c.initialized && method != "initialize" {
		return nil, fmt.Errorf("client not initialized")
	}

	id := c.requestID.Add(1)

	request := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSPNRPCVersion,
		Id:      id,
		Method:  method,
		Params:  params,
	}

	response, err := c.transport.SendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("transport error: %w", err)
	}

	// 检查错误（这里不需要检查 Error 字段，因为 JSONRPCResponse 没有这个字段）
	// 如果需要检查错误，应该在 transport.Interface 中处理

	// 将response.Result转换为json.RawMessage
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response result: %w", err)
	}
	return resultBytes, nil
}

// Initialize negotiates with the server.
// Must be called after Start, and before any request methods.
func (c *Client) Initialize(
	ctx context.Context,
	request mcp.InitializeRequestSchema,
) (*mcp.InitializedResult, error) {
	// 从 request 中提取参数
	var params mcp.InitializeRequestParams
	err := json.Unmarshal(request.Params, &params)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal initialize params: %w", err)
	}

	// 重新封装参数以确保格式正确
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	response, err := c.sendRequest(ctx, "initialize", paramsBytes)
	if err != nil {
		return nil, err
	}

	var result mcp.InitializeResult
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Store serverCapabilities
	c.serverCapabilities = result.Result.Capabilities

	// Send initialized notification
	notification := mcp.JSONRPCNotification{
		JSONRPCBaseResult: mcp.JSONRPCBaseResult{
			JSONRPC: mcp.JSPNRPCVersion,
			ID:      nil, // 通知不需要 ID
		},
		Method: "notifications/initialized",
	}

	err = c.transport.SendNotification(ctx, notification)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to send initialized notification: %w",
			err,
		)
	}

	c.initialized = true
	return &result.Result, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.sendRequest(ctx, "ping", nil)
	return err
}

func (c *Client) ListToolsByPage(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	result, err := listByPage[mcp.ListToolsResult](ctx, c, request.PaginatedRequest, "tools/list")
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListTools(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	result, err := c.ListToolsByPage(ctx, request)
	if err != nil {
		return nil, err
	}

	// 由于 ListToolsResult 没有 NextCursor 字段，下面的分页逻辑暂时注释掉
	// 如果需要分页功能，应该在 ListToolsResult 中添加相应字段
	/*
		for result.NextCursor != "" {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				request.Params.Cursor = result.NextCursor
				newPageRes, err := c.ListToolsByPage(ctx, request)
				if err != nil {
					return nil, err
				}
				result.Tools = append(result.Tools, newPageRes.Tools...)
				result.NextCursor = newPageRes.NextCursor
			}
		}
	*/

	return result, nil
}

func (c *Client) CallTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	paramsBytes, err := json.Marshal(request.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	response, err := c.sendRequest(ctx, "tools/call", paramsBytes)
	if err != nil {
		return nil, err
	}

	return mcp.ParseCallToolResult(&response)
}

func listByPage[T any](
	ctx context.Context,
	client *Client,
	request mcp.PaginatedRequest,
	method string,
) (*T, error) {
	paramsBytes, err := json.Marshal(request.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	response, err := client.sendRequest(ctx, method, paramsBytes)
	if err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetTransport returns the transport used by the client
func (c *Client) GetTransport() transport.Interface {
	return c.transport
}

// GetServerCapabilities returns the server capabilities.
func (c *Client) GetServerCapabilities() mcp.ServerCapabilitiesSchema {
	return c.serverCapabilities
}

// GetClientCapabilities returns the client capabilities.
func (c *Client) GetClientCapabilities() mcp.ClientCapabilitiesSchema {
	return c.clientCapabilities
}

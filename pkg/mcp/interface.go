package mcp

import "encoding/json"

// JSONRPCRequest represents a JSON-RPC request that expects a response
type JSONRPCRequest struct {
	// JSONRPC version, must be "2.0"
	JSONRPC string `json:"jsonrpc"`
	// A uniquely identifying ID for a request in JSON-RPC
	ID int64 `json:"id"`
	// The method to be invoked
	Method string `json:"method"`
	// The parameters to be passed to the method
	Params any `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *JSONRPCError   `json:"error"`
}

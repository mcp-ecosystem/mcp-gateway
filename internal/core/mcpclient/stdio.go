package mcpclient

import (
	"context"
	"fmt"

	"github.com/mcp-ecosystem/mcp-gateway/internal/core/mcpclient/transport"
)

func NewStdioMCPClient(
	command string,
	env []string,
	args ...string,
) (*Client, error) {
	stdioTransport := transport.NewStdio(command, env, args...)
	err := stdioTransport.Start(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to start stdio transport: %w", err)
	}

	return NewClient(stdioTransport), nil
}

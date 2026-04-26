package mcp

import (
	"context"
	"encoding/json"

	"claudecode/internal/core"
)

// clientLike abstracts over stdio Client and SSEClient so the same tool
// adapter can route calls regardless of transport.
type clientLike interface {
	Start(ctx context.Context) error
	Stop() error
	ListTools() ([]MCPTool, error)
	CallTool(ctx context.Context, name string, args json.RawMessage) (string, error)
}

// mcpTool adapts a single MCPTool exposed by an MCP client to the core.Tool interface.
type mcpTool struct {
	client       clientLike
	prefixedName string
	upstreamName string
	desc         string
	schema       json.RawMessage
}

// newTool builds a core.Tool that proxies to the given MCP server tool.
// prefixedName must already be in the form "mcp__<server>__<tool>".
func newTool(client clientLike, prefixedName string, t MCPTool) core.Tool {
	return &mcpTool{
		client:       client,
		prefixedName: prefixedName,
		upstreamName: t.Name,
		desc:         t.Description,
		schema:       t.InputSchema,
	}
}

func (t *mcpTool) Name() string            { return t.prefixedName }
func (t *mcpTool) Description() string     { return t.desc }
func (t *mcpTool) Schema() json.RawMessage { return t.schema }

func (t *mcpTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	return t.client.CallTool(ctx, t.upstreamName, input)
}

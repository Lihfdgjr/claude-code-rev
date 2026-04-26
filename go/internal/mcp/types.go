package mcp

import "encoding/json"

// Config describes a single MCP server. Stdio (Command + Args) and HTTP/SSE
// (URL) transports are supported; Transport selects between them when both
// fields could match. Env supplies environment variables for stdio or extra
// headers for SSE.
type Config struct {
	Name      string
	Transport string // "stdio" (default) or "sse"
	Command   string
	Args      []string
	URL       string
	Env       map[string]string
}

// MCPTool is a tool descriptor returned by tools/list.
type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// jsonrpcRequest is the wire form of an outgoing JSON-RPC request.
type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonrpcNotification is an outgoing notification (no id, no response).
type jsonrpcNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonrpcResponse is the wire form of an incoming JSON-RPC response.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
	Method  string          `json:"method,omitempty"`
}

// jsonrpcError is the standard JSON-RPC error object.
type jsonrpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *jsonrpcError) Error() string {
	return e.Message
}

// initializeParams is sent in the initialize request.
type initializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      clientInfo             `json:"clientInfo"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// listToolsResult is the result envelope for tools/list.
type listToolsResult struct {
	Tools []MCPTool `json:"tools"`
}

// callToolParams is sent in tools/call.
type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// callToolResult is the result envelope for tools/call.
type callToolResult struct {
	Content []contentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

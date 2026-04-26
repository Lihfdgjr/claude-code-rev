package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SSEConfig configures a remote MCP HTTP/SSE endpoint.
type SSEConfig struct {
	URL     string
	Headers map[string]string
}

// SSEClient is a simplified HTTP/SSE transport for an MCP endpoint. Each
// JSON-RPC request is POSTed to the configured URL; the response body is
// either a single JSON object or an SSE stream whose first non-empty
// "data:" line is parsed as a JSON-RPC response.
type SSEClient struct {
	cfg SSEConfig

	http   *http.Client
	nextID atomic.Int64

	stopMu  sync.Mutex
	stopped bool
	cancel  context.CancelFunc
	ctx     context.Context
}

// NewSSE constructs a new SSEClient from cfg.
func NewSSE(cfg SSEConfig) *SSEClient {
	return &SSEClient{
		cfg:  cfg,
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

// Start performs the MCP initialize handshake against the remote endpoint.
func (c *SSEClient) Start(ctx context.Context) error {
	if strings.TrimSpace(c.cfg.URL) == "" {
		return errors.New("mcp sse: URL is required")
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())

	initCtx, cancel := context.WithTimeout(ctx, initializeTO)
	defer cancel()

	params := initializeParams{
		ProtocolVersion: protocolVersion,
		Capabilities:    map[string]interface{}{},
		ClientInfo:      clientInfo{Name: "claudecode-go", Version: "0.1"},
	}
	if _, err := c.request(initCtx, "initialize", params); err != nil {
		return fmt.Errorf("mcp sse: initialize: %w", err)
	}
	if err := c.notify(initCtx, "notifications/initialized", struct{}{}); err != nil {
		return fmt.Errorf("mcp sse: initialized notification: %w", err)
	}
	return nil
}

// Stop releases resources. Safe to call twice.
func (c *SSEClient) Stop() error {
	c.stopMu.Lock()
	defer c.stopMu.Unlock()
	if c.stopped {
		return nil
	}
	c.stopped = true
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

// ListTools returns all tools exported by the remote server.
func (c *SSEClient) ListTools() ([]MCPTool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callToolTO)
	defer cancel()
	raw, err := c.request(ctx, "tools/list", struct{}{})
	if err != nil {
		return nil, err
	}
	var out listToolsResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("mcp sse: decode tools/list: %w", err)
	}
	return out.Tools, nil
}

// CallTool invokes a tool by name on the remote server.
func (c *SSEClient) CallTool(ctx context.Context, name string, args json.RawMessage) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, callToolTO)
	defer cancel()
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	raw, err := c.request(cctx, "tools/call", callToolParams{Name: name, Arguments: args})
	if err != nil {
		return "", err
	}
	var res callToolResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("mcp sse: decode tools/call: %w", err)
	}
	var buf strings.Builder
	for i, b := range res.Content {
		if b.Type == "text" {
			if i > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString(b.Text)
		}
	}
	if res.IsError {
		return buf.String(), fmt.Errorf("mcp sse: tool %s reported error: %s", name, buf.String())
	}
	return buf.String(), nil
}

// request POSTs a JSON-RPC request and decodes the response body, which may be
// a single JSON object or an SSE stream containing one event with the response.
func (c *SSEClient) request(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	req := jsonrpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	raw, err := c.postJSON(ctx, req, true)
	if err != nil {
		return nil, fmt.Errorf("mcp sse: %s: %w", method, err)
	}
	if raw == nil {
		return nil, fmt.Errorf("mcp sse: %s: empty response", method)
	}

	var resp jsonrpcResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("mcp sse: %s: decode: %w", method, err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("mcp sse: %s: %s (code %d)", method, resp.Error.Message, resp.Error.Code)
	}
	return resp.Result, nil
}

// notify POSTs a JSON-RPC notification with no response handling.
func (c *SSEClient) notify(ctx context.Context, method string, params interface{}) error {
	n := jsonrpcNotification{JSONRPC: "2.0", Method: method, Params: params}
	_, err := c.postJSON(ctx, n, false)
	if err != nil {
		return fmt.Errorf("mcp sse: notify %s: %w", method, err)
	}
	return nil
}

// postJSON sends v as a JSON body to the configured endpoint. When wantBody is
// true the response is parsed (either a JSON-RPC object directly or the first
// non-empty "data:" line of an SSE stream).
func (c *SSEClient) postJSON(ctx context.Context, v interface{}, wantBody bool) (json.RawMessage, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json, text/event-stream")
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	if !wantBody {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, nil
	}

	ct := strings.ToLower(resp.Header.Get("content-type"))
	switch {
	case strings.Contains(ct, "text/event-stream"):
		return readFirstSSEEvent(resp.Body)
	default:
		return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	}
}

// readFirstSSEEvent consumes the body until it finds the first non-empty
// "data:" line and returns its payload.
func readFirstSSEEvent(r io.Reader) (json.RawMessage, error) {
	br := bufio.NewReaderSize(r, 1<<16)
	var buf bytes.Buffer
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			line = bytes.TrimRight(line, "\r\n")
			if len(line) == 0 {
				if buf.Len() > 0 {
					return json.RawMessage(append([]byte(nil), buf.Bytes()...)), nil
				}
				continue
			}
			if bytes.HasPrefix(line, []byte("data:")) {
				payload := bytes.TrimSpace(line[len("data:"):])
				if buf.Len() > 0 {
					buf.WriteByte('\n')
				}
				buf.Write(payload)
			}
			// other fields (id:, event:, retry:) are ignored for this transport.
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if buf.Len() > 0 {
					return json.RawMessage(append([]byte(nil), buf.Bytes()...)), nil
				}
				return nil, io.EOF
			}
			return nil, err
		}
	}
}

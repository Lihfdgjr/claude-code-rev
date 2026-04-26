package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

const (
	protocolVersion = "2024-11-05"
	initializeTO    = 10 * time.Second
	callToolTO      = 30 * time.Second
)

// Client speaks JSON-RPC 2.0 to a single MCP stdio subprocess.
type Client struct {
	cfg Config

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	writeMu sync.Mutex

	nextID  atomic.Int64
	pendMu  sync.Mutex
	pending map[int64]chan *jsonrpcResponse

	doneCh chan struct{}
	stopMu sync.Mutex
	stopped bool
}

// New constructs a Client from the given config. Call Start to spawn it.
func New(cfg Config) *Client {
	return &Client{
		cfg:     cfg,
		pending: make(map[int64]chan *jsonrpcResponse),
		doneCh:  make(chan struct{}),
	}
}

// Start launches the subprocess, performs the MCP initialize handshake, and
// sends the initialized notification.
func (c *Client) Start(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.cfg.Command, c.cfg.Args...)
	if len(c.cfg.Env) > 0 {
		env := os.Environ()
		for k, v := range c.cfg.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("mcp %s: stdin pipe: %w", c.cfg.Name, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("mcp %s: stdout pipe: %w", c.cfg.Name, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mcp %s: start %s: %w", c.cfg.Name, c.cfg.Command, err)
	}

	c.cmd = cmd
	c.stdin = stdin
	c.stdout = stdout

	go c.readLoop()

	initCtx, cancel := context.WithTimeout(ctx, initializeTO)
	defer cancel()
	params := initializeParams{
		ProtocolVersion: protocolVersion,
		Capabilities:    map[string]interface{}{},
		ClientInfo:      clientInfo{Name: "claudecode-go", Version: "0.1"},
	}
	if _, err := c.request(initCtx, "initialize", params); err != nil {
		_ = c.Stop()
		return fmt.Errorf("mcp %s: initialize: %w", c.cfg.Name, err)
	}
	if err := c.notify("notifications/initialized", struct{}{}); err != nil {
		_ = c.Stop()
		return fmt.Errorf("mcp %s: initialized notification: %w", c.cfg.Name, err)
	}
	return nil
}

// Stop terminates the subprocess and releases resources. Safe to call twice.
func (c *Client) Stop() error {
	c.stopMu.Lock()
	if c.stopped {
		c.stopMu.Unlock()
		return nil
	}
	c.stopped = true
	c.stopMu.Unlock()

	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}
	return nil
}

// ListTools returns all tools exported by the server.
func (c *Client) ListTools() ([]MCPTool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callToolTO)
	defer cancel()
	raw, err := c.request(ctx, "tools/list", struct{}{})
	if err != nil {
		return nil, err
	}
	var out listToolsResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("mcp %s: decode tools/list: %w", c.cfg.Name, err)
	}
	return out.Tools, nil
}

// CallTool invokes a tool by upstream name and returns concatenated text content.
func (c *Client) CallTool(ctx context.Context, name string, args json.RawMessage) (string, error) {
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
		return "", fmt.Errorf("mcp %s: decode tools/call: %w", c.cfg.Name, err)
	}
	var buf string
	for i, b := range res.Content {
		if b.Type == "text" {
			if i > 0 {
				buf += "\n"
			}
			buf += b.Text
		}
	}
	if res.IsError {
		return buf, fmt.Errorf("mcp %s: tool %s reported error: %s", c.cfg.Name, name, buf)
	}
	return buf, nil
}

// request sends a JSON-RPC request and waits for its matching response.
func (c *Client) request(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	ch := make(chan *jsonrpcResponse, 1)
	c.pendMu.Lock()
	c.pending[id] = ch
	c.pendMu.Unlock()
	defer func() {
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
	}()

	req := jsonrpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	if err := c.writeJSON(req); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("mcp %s: %s: %w", c.cfg.Name, method, ctx.Err())
	case <-c.doneCh:
		return nil, fmt.Errorf("mcp %s: %s: subprocess exited", c.cfg.Name, method)
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("mcp %s: %s: %s (code %d)", c.cfg.Name, method, resp.Error.Message, resp.Error.Code)
		}
		return resp.Result, nil
	}
}

// notify sends a JSON-RPC notification with no expected response.
func (c *Client) notify(method string, params interface{}) error {
	n := jsonrpcNotification{JSONRPC: "2.0", Method: method, Params: params}
	return c.writeJSON(n)
}

// writeJSON serialises v as a single newline-delimited JSON message.
func (c *Client) writeJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.stdin == nil {
		return errors.New("mcp client: stdin closed")
	}
	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

// readLoop reads responses from the subprocess and dispatches them to waiters.
func (c *Client) readLoop() {
	defer close(c.doneCh)
	r := bufio.NewReaderSize(c.stdout, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			c.dispatch(line)
		}
		if err != nil {
			return
		}
	}
}

// dispatch decodes one line and routes it to the matching pending request.
func (c *Client) dispatch(line []byte) {
	var resp jsonrpcResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return
	}
	if resp.ID == nil {
		// Notification from server: discard for now.
		return
	}
	c.pendMu.Lock()
	ch, ok := c.pending[*resp.ID]
	c.pendMu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- &resp:
	default:
	}
}

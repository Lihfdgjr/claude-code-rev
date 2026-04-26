package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client is the language-server surface used by tools.
type Client interface {
	Start(ctx context.Context, command string, args []string) error
	Stop() error
	Definition(ctx context.Context, file string, line, col int) (string, error)
	Hover(ctx context.Context, file string, line, col int) (string, error)
	References(ctx context.Context, file string, line, col int) ([]string, error)
	Symbols(ctx context.Context, file string) ([]string, error)
}

// ErrNotStarted is returned when a method is called before Start succeeds.
var ErrNotStarted = errors.New("lsp: client not started")

// New returns a fresh JSON-RPC LSP client.
func New() Client { return &client{} }

type client struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	nextID  atomic.Int64
	mu      sync.Mutex
	pending map[int64]chan rpcResp
	started bool
	opened  map[string]bool
	writeMu sync.Mutex
}

type rpcReq struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcNotif struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcErr         `json:"error,omitempty"`
}

// --- lifecycle ---------------------------------------------------------------

func (c *client) Start(ctx context.Context, command string, args []string) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("lsp: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("lsp: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("lsp: start %q: %w", command, err)
	}

	c.mu.Lock()
	c.cmd = cmd
	c.stdin = stdin
	c.stdout = stdout
	c.pending = make(map[int64]chan rpcResp)
	c.opened = make(map[string]bool)
	c.mu.Unlock()

	go c.readLoop()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	rootURI := fileURI(cwd)
	initParams := map[string]interface{}{
		"processId":    os.Getpid(),
		"rootUri":      rootURI,
		"capabilities": map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "claudecode",
			"version": "0.1.0",
		},
	}

	initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := c.call(initCtx, "initialize", initParams); err != nil {
		_ = c.cmd.Process.Kill()
		return fmt.Errorf("lsp: initialize: %w", err)
	}
	if err := c.notify("initialized", map[string]interface{}{}); err != nil {
		return fmt.Errorf("lsp: initialized: %w", err)
	}

	c.mu.Lock()
	c.started = true
	c.mu.Unlock()
	return nil
}

// Started reports whether Start has completed successfully.
func (c *client) Started() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.started
}

func (c *client) Stop() error {
	c.mu.Lock()
	started := c.started
	cmd := c.cmd
	c.mu.Unlock()
	if !started || cmd == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = c.call(ctx, "shutdown", nil)
	_ = c.notify("exit", nil)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		return <-done
	}
}

// --- read loop & framing -----------------------------------------------------

func (c *client) readLoop() {
	r := bufio.NewReader(c.stdout)
	for {
		body, err := readMessage(r)
		if err != nil {
			c.mu.Lock()
			for id, ch := range c.pending {
				close(ch)
				delete(c.pending, id)
			}
			c.mu.Unlock()
			return
		}
		var resp rpcResp
		if err := json.Unmarshal(body, &resp); err != nil || resp.ID == 0 {
			continue
		}
		c.mu.Lock()
		ch, ok := c.pending[resp.ID]
		c.mu.Unlock()
		if ok {
			ch <- resp
		}
	}
}

func readMessage(r *bufio.Reader) ([]byte, error) {
	var contentLength int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if i := strings.IndexByte(line, ':'); i >= 0 {
			key := strings.TrimSpace(line[:i])
			val := strings.TrimSpace(line[i+1:])
			if strings.EqualFold(key, "Content-Length") {
				n, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("invalid Content-Length: %q", val)
				}
				contentLength = n
			}
		}
	}
	if contentLength <= 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) writeFrame(body []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(c.stdin, header); err != nil {
		return err
	}
	_, err := c.stdin.Write(body)
	return err
}

// --- request helpers ---------------------------------------------------------

func (c *client) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	ch := make(chan rpcResp, 1)
	c.mu.Lock()
	if c.pending == nil {
		c.mu.Unlock()
		return nil, ErrNotStarted
	}
	c.pending[id] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	req := rpcReq{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := c.writeFrame(body); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp, ok := <-ch:
		if !ok {
			return nil, io.ErrUnexpectedEOF
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("lsp %s: %s (code %d)", method, resp.Error.Message, resp.Error.Code)
		}
		return resp.Result, nil
	}
}

func (c *client) notify(method string, params interface{}) error {
	n := rpcNotif{JSONRPC: "2.0", Method: method, Params: params}
	body, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return c.writeFrame(body)
}

// --- document sync -----------------------------------------------------------

var languageIDs = map[string]string{
	".go":   "go",
	".py":   "python",
	".js":   "javascript",
	".jsx":  "javascriptreact",
	".ts":   "typescript",
	".tsx":  "typescriptreact",
	".rs":   "rust",
	".rb":   "ruby",
	".java": "java",
	".c":    "c",
	".h":    "c",
	".cc":   "cpp",
	".cpp":  "cpp",
	".hpp":  "cpp",
	".cs":   "csharp",
	".php":  "php",
	".sh":   "shellscript",
	".json": "json",
	".yaml": "yaml",
	".yml":  "yaml",
	".md":   "markdown",
}

func languageID(path string) string {
	if id, ok := languageIDs[strings.ToLower(filepath.Ext(path))]; ok {
		return id
	}
	return "plaintext"
}

func (c *client) ensureOpen(file string) error {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return ErrNotStarted
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		c.mu.Unlock()
		return err
	}
	if c.opened[abs] {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        fileURI(abs),
			"languageId": languageID(abs),
			"version":    1,
			"text":       string(data),
		},
	}
	if err := c.notify("textDocument/didOpen", params); err != nil {
		return err
	}
	c.mu.Lock()
	c.opened[abs] = true
	c.mu.Unlock()
	return nil
}

// --- queries -----------------------------------------------------------------

type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

type lspLocation struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

func (c *client) docPosParams(file string, line, col int) (map[string]interface{}, string, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return nil, "", err
	}
	return map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": fileURI(abs)},
		"position":     map[string]interface{}{"line": line, "character": col},
	}, abs, nil
}

func (c *client) Definition(ctx context.Context, file string, line, col int) (string, error) {
	if err := c.ensureOpen(file); err != nil {
		return "", err
	}
	params, _, err := c.docPosParams(file, line, col)
	if err != nil {
		return "", err
	}
	raw, err := c.call(ctx, "textDocument/definition", params)
	if err != nil {
		return "", err
	}
	locs, err := decodeLocations(raw)
	if err != nil {
		return "", err
	}
	if len(locs) == 0 {
		return "", nil
	}
	return formatLocation(locs[0]), nil
}

func (c *client) Hover(ctx context.Context, file string, line, col int) (string, error) {
	if err := c.ensureOpen(file); err != nil {
		return "", err
	}
	params, _, err := c.docPosParams(file, line, col)
	if err != nil {
		return "", err
	}
	raw, err := c.call(ctx, "textDocument/hover", params)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var hover struct {
		Contents json.RawMessage `json:"contents"`
	}
	if err := json.Unmarshal(raw, &hover); err != nil {
		return "", err
	}
	value := decodeHoverContents(hover.Contents)
	if len(value) > 2000 {
		value = value[:2000]
	}
	return value, nil
}

func (c *client) References(ctx context.Context, file string, line, col int) ([]string, error) {
	if err := c.ensureOpen(file); err != nil {
		return nil, err
	}
	params, _, err := c.docPosParams(file, line, col)
	if err != nil {
		return nil, err
	}
	params["context"] = map[string]interface{}{"includeDeclaration": true}
	raw, err := c.call(ctx, "textDocument/references", params)
	if err != nil {
		return nil, err
	}
	locs, err := decodeLocations(raw)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(locs))
	for _, l := range locs {
		out = append(out, formatLocation(l))
	}
	return out, nil
}

func (c *client) Symbols(ctx context.Context, file string) ([]string, error) {
	if err := c.ensureOpen(file); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": fileURI(abs)},
	}
	raw, err := c.call(ctx, "textDocument/documentSymbol", params)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var docSyms []struct {
		Name     string `json:"name"`
		Kind     int    `json:"kind"`
		Children []struct {
			Name string `json:"name"`
			Kind int    `json:"kind"`
		} `json:"children"`
	}
	if err := json.Unmarshal(raw, &docSyms); err == nil && len(docSyms) > 0 && docSyms[0].Name != "" {
		var out []string
		for _, s := range docSyms {
			out = append(out, fmt.Sprintf("%s %s", symbolKind(s.Kind), s.Name))
			for _, c := range s.Children {
				out = append(out, fmt.Sprintf("%s %s.%s", symbolKind(c.Kind), s.Name, c.Name))
			}
		}
		return out, nil
	}
	var symInfo []struct {
		Name string `json:"name"`
		Kind int    `json:"kind"`
	}
	if err := json.Unmarshal(raw, &symInfo); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(symInfo))
	for _, s := range symInfo {
		out = append(out, fmt.Sprintf("%s %s", symbolKind(s.Kind), s.Name))
	}
	return out, nil
}

// --- decoding helpers --------------------------------------------------------

func decodeLocations(raw json.RawMessage) ([]lspLocation, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	if raw[0] == '[' {
		var locs []lspLocation
		if err := json.Unmarshal(raw, &locs); err == nil {
			return locs, nil
		}
		var links []struct {
			TargetURI   string   `json:"targetUri"`
			TargetRange lspRange `json:"targetRange"`
		}
		if err := json.Unmarshal(raw, &links); err != nil {
			return nil, err
		}
		out := make([]lspLocation, 0, len(links))
		for _, l := range links {
			out = append(out, lspLocation{URI: l.TargetURI, Range: l.TargetRange})
		}
		return out, nil
	}
	var loc lspLocation
	if err := json.Unmarshal(raw, &loc); err != nil {
		return nil, err
	}
	return []lspLocation{loc}, nil
}

func decodeHoverContents(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var mc struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &mc); err == nil && mc.Value != "" {
		return mc.Value
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		var b strings.Builder
		for i, item := range arr {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(decodeHoverContents(item))
		}
		return b.String()
	}
	return ""
}

func formatLocation(l lspLocation) string {
	return fmt.Sprintf("%s:%d:%d", uriToPath(l.URI), l.Range.Start.Line+1, l.Range.Start.Character+1)
}

func symbolKind(k int) string {
	names := map[int]string{
		1: "File", 2: "Module", 3: "Namespace", 4: "Package", 5: "Class",
		6: "Method", 7: "Property", 8: "Field", 9: "Constructor", 10: "Enum",
		11: "Interface", 12: "Function", 13: "Variable", 14: "Constant",
		15: "String", 16: "Number", 17: "Boolean", 18: "Array", 19: "Object",
		20: "Key", 21: "Null", 22: "EnumMember", 23: "Struct", 24: "Event",
		25: "Operator", 26: "TypeParameter",
	}
	if n, ok := names[k]; ok {
		return n
	}
	return "Symbol"
}

// --- URI helpers -------------------------------------------------------------

// fileURI converts an absolute filesystem path to a file:// URI. On Windows
// it produces forms like file:///C:/foo/bar.
func fileURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.ToSlash(abs)
	if len(abs) >= 2 && abs[1] == ':' {
		return "file:///" + abs
	}
	if !strings.HasPrefix(abs, "/") {
		abs = "/" + abs
	}
	return "file://" + abs
}

func uriToPath(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return uri
	}
	rest := strings.TrimPrefix(uri, "file://")
	if decoded, err := url.PathUnescape(rest); err == nil {
		rest = decoded
	}
	if len(rest) >= 3 && rest[0] == '/' && rest[2] == ':' {
		rest = rest[1:]
	}
	return filepath.FromSlash(rest)
}

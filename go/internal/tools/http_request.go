package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"claudecode/internal/core"
)

type httpRequestTool struct{}

type httpRequestInput struct {
	URL       string            `json:"url"`
	Method    string            `json:"method,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	TimeoutMS int               `json:"timeout_ms,omitempty"`
}

func NewHTTPRequest() core.Tool { return &httpRequestTool{} }

func (httpRequestTool) Name() string { return "HTTPRequest" }

func (httpRequestTool) Description() string {
	return "Issue an HTTP request with arbitrary method, headers, and body. Returns status, response headers, and body (capped at 50000 chars)."
}

func (httpRequestTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "url": {"type": "string"},
    "method": {"type": "string", "enum": ["GET", "POST", "PUT", "DELETE", "PATCH"]},
    "headers": {"type": "object", "additionalProperties": {"type": "string"}},
    "body": {"type": "string"},
    "timeout_ms": {"type": "integer", "minimum": 1}
  },
  "required": ["url"],
  "additionalProperties": false
}`)
}

func (httpRequestTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in httpRequestInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.URL == "" {
		return "", fmt.Errorf("url is required")
	}
	method := strings.ToUpper(in.Method)
	if method == "" {
		method = "GET"
	}
	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH":
	default:
		return "", fmt.Errorf("unsupported method: %s", method)
	}

	timeout := time.Duration(in.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	rctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var body io.Reader
	if in.Body != "" {
		body = bytes.NewBufferString(in.Body)
	}
	req, err := http.NewRequestWithContext(rctx, method, in.URL, body)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	for k, v := range in.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	text := string(respBody)
	if len(text) > 50000 {
		text = text[:50000] + "\n...[truncated to 50000 chars]"
	}

	keys := make([]string, 0, len(resp.Header))
	for k := range resp.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	fmt.Fprintf(&b, "status %d\n", resp.StatusCode)
	for _, k := range keys {
		for _, v := range resp.Header.Values(k) {
			fmt.Fprintf(&b, "%s: %s\n", k, v)
		}
	}
	b.WriteString("\n")
	b.WriteString(text)
	return b.String(), nil
}

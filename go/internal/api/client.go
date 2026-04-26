package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"claudecode/internal/core"
)

const (
	defaultBaseURL    = "https://api.anthropic.com"
	defaultMaxTokens  = 8192
	defaultUserAgent  = "claudecode-go/0.1"
	anthropicVersion  = "2023-06-01"
	streamChannelSize = 16
)

type Options struct {
	APIKey           string
	BaseURL          string
	HTTPClient       *http.Client
	DefaultMaxTokens int
	UserAgent        string
}

type Client struct {
	apiKey     string
	baseURL    string
	http       *http.Client
	maxTokens  int
	userAgent  string
}

func New(opts Options) *Client {
	c := &Client{
		apiKey:    opts.APIKey,
		baseURL:   strings.TrimRight(opts.BaseURL, "/"),
		http:      opts.HTTPClient,
		maxTokens: opts.DefaultMaxTokens,
		userAgent: opts.UserAgent,
	}
	if c.baseURL == "" {
		c.baseURL = defaultBaseURL
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: 0}
	}
	if c.maxTokens <= 0 {
		c.maxTokens = defaultMaxTokens
	}
	if c.userAgent == "" {
		c.userAgent = defaultUserAgent
	}
	return c
}

// Stream implements core.Transport.
func (c *Client) Stream(ctx context.Context, opts core.CallOptions, history []core.Message) (<-chan core.StreamEvent, error) {
	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	body := Request{
		Model:       opts.Model,
		System:      opts.SystemPrompt,
		Messages:    messagesFromCore(history),
		Tools:       toolsFromCore(opts.Tools),
		MaxTokens:   maxTokens,
		Temperature: opts.Temperature,
		Stream:      true,
	}
	if opts.Thinking {
		body.Thinking = &Thinking{Type: "enabled", BudgetTokens: 8000}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("api: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("api: build request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "text/event-stream")
	req.Header.Set("user-agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api: http do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := readBodySnippet(resp.Body, 2048)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("api: http %d %s: %s", resp.StatusCode, resp.Status, snippet)
	}

	out := make(chan core.StreamEvent, streamChannelSize)
	go c.pump(ctx, resp, out)
	return out, nil
}

func (c *Client) pump(ctx context.Context, resp *http.Response, out chan<- core.StreamEvent) {
	defer resp.Body.Close()
	defer close(out)

	reader := newSSEReader(resp.Body)

	var (
		stopReason     string
		usage          core.Usage
		blockCitations = map[int][]core.Citation{}
	)
	// blockCitations is tracked alongside text blocks for future incremental
	// delivery; the streaming path does not yet emit citation events. The
	// non-streaming wire conversion is responsible for attaching them.
	_ = blockCitations

	send := func(ev core.StreamEvent) bool {
		select {
		case <-ctx.Done():
			return false
		case out <- ev:
			return true
		}
	}

	for {
		// Cooperative cancellation between events.
		select {
		case <-ctx.Done():
			return
		default:
		}

		ev, err := reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			if ctx.Err() != nil {
				return
			}
			send(core.ErrorEvent{Err: fmt.Errorf("api: sse read: %w", err)})
			return
		}

		switch ev.Name {
		case "message_start":
			var ms sseMessageStart
			if err := json.Unmarshal([]byte(ev.Data), &ms); err != nil {
				send(core.ErrorEvent{Err: fmt.Errorf("api: decode message_start: %w", err)})
				return
			}
			usage.InputTokens += ms.Message.Usage.InputTokens
			usage.OutputTokens += ms.Message.Usage.OutputTokens
			usage.CacheReadTokens += ms.Message.Usage.CacheReadInputTokens
			usage.CacheCreationTokens += ms.Message.Usage.CacheCreationInputTokens
			if !send(core.MessageStartEvent{ID: ms.Message.ID, Model: ms.Message.Model}) {
				return
			}

		case "content_block_start":
			var cbs sseContentBlockStart
			if err := json.Unmarshal([]byte(ev.Data), &cbs); err != nil {
				send(core.ErrorEvent{Err: fmt.Errorf("api: decode content_block_start: %w", err)})
				return
			}
			switch cbs.ContentBlock.Type {
			case "tool_use":
				if !send(core.ToolUseStartEvent{
					Index: cbs.Index,
					ID:    cbs.ContentBlock.ID,
					Name:  cbs.ContentBlock.Name,
				}) {
					return
				}
			case "text":
				if len(cbs.ContentBlock.Citations) > 0 {
					blockCitations[cbs.Index] = wireCitationsToCore(cbs.ContentBlock.Citations)
				}
			}

		case "content_block_delta":
			var cbd sseContentBlockDelta
			if err := json.Unmarshal([]byte(ev.Data), &cbd); err != nil {
				send(core.ErrorEvent{Err: fmt.Errorf("api: decode content_block_delta: %w", err)})
				return
			}
			switch cbd.Delta.Type {
			case "text_delta":
				if !send(core.TextDeltaEvent{Index: cbd.Index, Text: cbd.Delta.Text}) {
					return
				}
			case "input_json_delta":
				if !send(core.ToolInputDeltaEvent{Index: cbd.Index, JSONPart: cbd.Delta.PartialJSON}) {
					return
				}
			case "thinking_delta":
				if !send(core.ThinkingDeltaEvent{Index: cbd.Index, Text: cbd.Delta.Thinking}) {
					return
				}
			case "signature_delta":
				// Not surfaced through core stream events.
			}

		case "content_block_stop":
			var cbst sseContentBlockStop
			if err := json.Unmarshal([]byte(ev.Data), &cbst); err != nil {
				send(core.ErrorEvent{Err: fmt.Errorf("api: decode content_block_stop: %w", err)})
				return
			}
			if !send(core.BlockEndEvent{Index: cbst.Index}) {
				return
			}

		case "message_delta":
			var md sseMessageDelta
			if err := json.Unmarshal([]byte(ev.Data), &md); err != nil {
				send(core.ErrorEvent{Err: fmt.Errorf("api: decode message_delta: %w", err)})
				return
			}
			if md.Delta.StopReason != "" {
				stopReason = md.Delta.StopReason
			}
			usage.InputTokens += md.Usage.InputTokens
			usage.OutputTokens += md.Usage.OutputTokens
			usage.CacheReadTokens += md.Usage.CacheReadInputTokens
			usage.CacheCreationTokens += md.Usage.CacheCreationInputTokens

		case "message_stop":
			send(core.MessageEndEvent{StopReason: stopReason, Usage: usage})
			return

		case "error":
			var e sseError
			if err := json.Unmarshal([]byte(ev.Data), &e); err != nil {
				send(core.ErrorEvent{Err: fmt.Errorf("api: decode error: %w", err)})
				return
			}
			send(core.ErrorEvent{Err: fmt.Errorf("api: %s: %s", e.Error.Type, e.Error.Message)})
			return

		case "ping", "":
			// Ignore.
		}
	}
}

func readBodySnippet(r io.Reader, max int) string {
	if r == nil {
		return ""
	}
	buf := make([]byte, max)
	n, _ := io.ReadFull(io.LimitReader(r, int64(max)), buf)
	// Drain remainder so the connection can be reused.
	_, _ = io.Copy(io.Discard, r)
	return strings.TrimSpace(string(buf[:n]))
}

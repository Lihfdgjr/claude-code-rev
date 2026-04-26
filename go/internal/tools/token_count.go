package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"claudecode/internal/core"
)

type tokenCountTool struct{}

type tokenCountInput struct {
	Text   string `json:"text"`
	Model  string `json:"model,omitempty"`
	System string `json:"system,omitempty"`
}

func NewTokenCount() core.Tool { return &tokenCountTool{} }

func (tokenCountTool) Name() string { return "TokenCount" }

func (tokenCountTool) Description() string {
	return "Count tokens for a string against the Anthropic /v1/messages/count_tokens endpoint. Falls back to a 4-chars-per-token estimate if ANTHROPIC_API_KEY is unset."
}

func (tokenCountTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "text": {"type": "string"},
    "model": {"type": "string", "description": "Optional model id; defaults to env CLAUDECODE_MODEL or claude-opus-4-7"},
    "system": {"type": "string", "description": "Optional system prompt to include in the count"}
  },
  "required": ["text"],
  "additionalProperties": false
}`)
}

const defaultTokenCountModel = "claude-opus-4-7"

type ctReqContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ctReqMessage struct {
	Role    string         `json:"role"`
	Content []ctReqContent `json:"content"`
}

type ctRequest struct {
	Model    string         `json:"model"`
	System   string         `json:"system,omitempty"`
	Messages []ctReqMessage `json:"messages"`
}

type ctResponse struct {
	InputTokens int `json:"input_tokens"`
}

func resolveModel(in string) string {
	if in != "" {
		return in
	}
	if env := os.Getenv("CLAUDECODE_MODEL"); env != "" {
		return env
	}
	return defaultTokenCountModel
}

func roughEstimate(text string) int { return (len(text) + 3) / 4 }

func (tokenCountTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in tokenCountInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	model := resolveModel(in.Model)
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		n := roughEstimate(in.Text)
		return fmt.Sprintf("~%d input tokens (model=%s, rough estimate; ANTHROPIC_API_KEY not set)", n, model), nil
	}

	body := ctRequest{
		Model:  model,
		System: in.System,
		Messages: []ctReqMessage{{
			Role:    "user",
			Content: []ctReqContent{{Type: "text", Text: in.Text}},
		}},
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.anthropic.com/v1/messages/count_tokens", bytes.NewReader(buf))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("count_tokens: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("count_tokens %s: %s", resp.Status, string(raw))
	}
	var parsed ctResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return fmt.Sprintf("%d input tokens (model=%s)", parsed.InputTokens, model), nil
}

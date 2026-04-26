package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"claudecode/internal/core"
)

const (
	batchSubmitEndpoint = "https://api.anthropic.com/v1/messages/batches"
	batchSubmitBeta     = "message-batches-2024-09-24"
)

type batchSubmitTool struct{}

// NewBatchSubmit returns a Tool that submits a Messages API batch and returns
// the batch id and status.
func NewBatchSubmit() core.Tool { return &batchSubmitTool{} }

func (t *batchSubmitTool) Name() string { return "BatchSubmit" }

func (t *batchSubmitTool) Description() string {
	return "Submit a Messages API batch (one or more requests with custom_id + params). Returns batch id and status. Requires ANTHROPIC_API_KEY in env."
}

func (t *batchSubmitTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"requests": {
				"type": "array",
				"minItems": 1,
				"items": {
					"type": "object",
					"properties": {
						"custom_id": {"type": "string"},
						"params": {"type": "object", "description": "A Messages API request body."}
					},
					"required": ["custom_id", "params"],
					"additionalProperties": false
				}
			}
		},
		"required": ["requests"],
		"additionalProperties": false
	}`)
}

type batchRequestItem struct {
	CustomID string          `json:"custom_id"`
	Params   json.RawMessage `json:"params"`
}

type batchSubmitRequest struct {
	Requests []batchRequestItem `json:"requests"`
}

func (t *batchSubmitTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args batchSubmitRequest
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if len(args.Requests) == 0 {
		return "", fmt.Errorf("requests must be a non-empty array")
	}
	for i, r := range args.Requests {
		if strings.TrimSpace(r.CustomID) == "" {
			return "", fmt.Errorf("requests[%d].custom_id is required", i)
		}
		if len(r.Params) == 0 {
			return "", fmt.Errorf("requests[%d].params is required", i)
		}
	}

	apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}

	body, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, batchSubmitEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthropicVersionHdr)
	req.Header.Set("anthropic-beta", batchSubmitBeta)
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("batches api %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		ID                string `json:"id"`
		Type              string `json:"type"`
		ProcessingStatus  string `json:"processing_status"`
		RequestCounts     struct {
			Processing int `json:"processing"`
			Succeeded  int `json:"succeeded"`
			Errored    int `json:"errored"`
			Canceled   int `json:"canceled"`
			Expired    int `json:"expired"`
		} `json:"request_counts"`
		CreatedAt   string `json:"created_at"`
		ExpiresAt   string `json:"expires_at"`
		ResultsURL  string `json:"results_url"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if parsed.ID == "" {
		return "", fmt.Errorf("response missing id: %s", strings.TrimSpace(string(respBody)))
	}

	return fmt.Sprintf("batch_id=%s processing_status=%s submitted=%d", parsed.ID, parsed.ProcessingStatus, len(args.Requests)), nil
}

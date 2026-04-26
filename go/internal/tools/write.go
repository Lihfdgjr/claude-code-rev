package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"claudecode/internal/core"
)

type writeTool struct{}

type writeInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func NewWrite() core.Tool { return &writeTool{} }

func (writeTool) Name() string { return "Write" }

func (writeTool) Description() string {
	return "Write content to a file, creating parent directories as needed. Overwrites existing files."
}

func (writeTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "file_path": {"type": "string", "description": "Absolute path to the file"},
    "content": {"type": "string", "description": "Content to write"}
  },
  "required": ["file_path", "content"],
  "additionalProperties": false
}`)
}

func (writeTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in writeInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}
	if !filepath.IsAbs(in.FilePath) {
		return "", fmt.Errorf("file_path must be absolute")
	}

	dir := filepath.Dir(in.FilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	data := []byte(in.Content)
	if err := os.WriteFile(in.FilePath, data, 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(data), in.FilePath), nil
}

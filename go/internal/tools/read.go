package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claudecode/internal/core"
)

type readTool struct{}

type readInput struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

func NewRead() core.Tool { return &readTool{} }

func (readTool) Name() string { return "Read" }

func (readTool) Description() string {
	return "Read a text file from the local filesystem. Returns content prefixed with line numbers. Supports optional offset and limit for partial reads."
}

func (readTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "file_path": {"type": "string", "description": "Absolute path to the file"},
    "offset": {"type": "integer", "description": "Starting line number (1-based)", "minimum": 1},
    "limit": {"type": "integer", "description": "Maximum number of lines to read", "minimum": 1}
  },
  "required": ["file_path"],
  "additionalProperties": false
}`)
}

func (readTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in readInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}
	if !filepath.IsAbs(in.FilePath) {
		return "", fmt.Errorf("file_path must be absolute")
	}

	f, err := os.Open(in.FilePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	start := 1
	if in.Offset > 0 {
		start = in.Offset
	}
	limit := in.Limit
	if limit <= 0 {
		limit = -1
	}

	var b strings.Builder
	lineNo := 0
	emitted := 0
	for scanner.Scan() {
		lineNo++
		if lineNo < start {
			continue
		}
		if limit > 0 && emitted >= limit {
			break
		}
		fmt.Fprintf(&b, "%6d\t%s\n", lineNo, scanner.Text())
		emitted++
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return b.String(), nil
}

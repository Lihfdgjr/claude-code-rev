package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claudecode/internal/core"
)

type lsTool struct{}

type lsInput struct {
	Path string `json:"path"`
}

func NewLS() core.Tool { return &lsTool{} }

func (lsTool) Name() string { return "LS" }

func (lsTool) Description() string {
	return "List the entries of a directory. Directories are suffixed with a slash."
}

func (lsTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "path": {"type": "string", "description": "Absolute path to the directory"}
  },
  "required": ["path"],
  "additionalProperties": false
}`)
}

func (lsTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in lsInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(in.Path) {
		return "", fmt.Errorf("path must be absolute")
	}

	entries, err := os.ReadDir(in.Path)
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, "\n"), nil
}

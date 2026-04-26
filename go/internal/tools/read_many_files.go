package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claudecode/internal/core"
)

type readManyFilesTool struct{}

type readManyFilesInput struct {
	FilePaths       []string `json:"file_paths"`
	MaxCharsPerFile int      `json:"max_chars_per_file,omitempty"`
}

func NewReadManyFiles() core.Tool { return &readManyFilesTool{} }

func (readManyFilesTool) Name() string { return "ReadManyFiles" }

func (readManyFilesTool) Description() string {
	return "Read multiple files in one call. Each file's contents are concatenated with '=== <path> ===' separators. Per-file output is capped to max_chars_per_file (default 10000)."
}

func (readManyFilesTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "file_paths": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Absolute file paths"
    },
    "max_chars_per_file": {"type": "integer", "minimum": 1, "description": "Per-file character cap (default 10000)"}
  },
  "required": ["file_paths"],
  "additionalProperties": false
}`)
}

func (readManyFilesTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in readManyFilesInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if len(in.FilePaths) == 0 {
		return "", fmt.Errorf("file_paths is required")
	}
	maxChars := in.MaxCharsPerFile
	if maxChars <= 0 {
		maxChars = 10000
	}

	var b strings.Builder
	for _, p := range in.FilePaths {
		fmt.Fprintf(&b, "=== %s ===\n", p)
		if !filepath.IsAbs(p) {
			b.WriteString("[error: path must be absolute]\n\n")
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(&b, "[error: %v]\n\n", err)
			continue
		}
		s := string(data)
		if len(s) > maxChars {
			s = s[:maxChars] + "\n...[truncated]"
		}
		b.WriteString(s)
		if !strings.HasSuffix(s, "\n") {
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

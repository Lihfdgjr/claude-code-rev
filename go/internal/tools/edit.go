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

type editTool struct{}

type editInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

func NewEdit() core.Tool { return &editTool{} }

func (editTool) Name() string { return "Edit" }

func (editTool) Description() string {
	return "Replace occurrences of old_string with new_string in a file. By default the match must be unique; set replace_all to substitute every occurrence."
}

func (editTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "file_path": {"type": "string", "description": "Absolute path to the file"},
    "old_string": {"type": "string", "description": "Exact text to replace"},
    "new_string": {"type": "string", "description": "Replacement text"},
    "replace_all": {"type": "boolean", "description": "Replace all occurrences", "default": false}
  },
  "required": ["file_path", "old_string", "new_string"],
  "additionalProperties": false
}`)
}

func (editTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in editInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}
	if !filepath.IsAbs(in.FilePath) {
		return "", fmt.Errorf("file_path must be absolute")
	}
	if in.OldString == "" {
		return "", fmt.Errorf("old_string must not be empty")
	}
	if in.OldString == in.NewString {
		return "", fmt.Errorf("old_string and new_string must differ")
	}

	data, err := os.ReadFile(in.FilePath)
	if err != nil {
		return "", err
	}
	content := string(data)
	count := strings.Count(content, in.OldString)
	if count == 0 {
		return "", fmt.Errorf("old_string not found in %s", in.FilePath)
	}
	if count > 1 && !in.ReplaceAll {
		return "", fmt.Errorf("old_string is not unique (%d occurrences); set replace_all to replace all", count)
	}

	var updated string
	var replaced int
	if in.ReplaceAll {
		updated = strings.ReplaceAll(content, in.OldString, in.NewString)
		replaced = count
	} else {
		updated = strings.Replace(content, in.OldString, in.NewString, 1)
		replaced = 1
	}

	if err := os.WriteFile(in.FilePath, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("edited %s (%d replacements)", in.FilePath, replaced), nil
}

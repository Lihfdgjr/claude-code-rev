package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"claudecode/internal/core"
)

type multiEditTool struct{}

func NewMultiEdit() core.Tool { return &multiEditTool{} }

func (t *multiEditTool) Name() string { return "MultiEdit" }

func (t *multiEditTool) Description() string {
	return "Apply a sequence of string-replacement edits to a single file in order. Aborts on the first edit whose old_string is not found."
}

func (t *multiEditTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {"type": "string"},
			"edits": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"old_string": {"type": "string"},
						"new_string": {"type": "string"},
						"replace_all": {"type": "boolean"}
					},
					"required": ["old_string", "new_string"],
					"additionalProperties": false
				}
			}
		},
		"required": ["file_path", "edits"],
		"additionalProperties": false
	}`)
}

func (t *multiEditTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		FilePath string `json:"file_path"`
		Edits    []struct {
			OldString  string `json:"old_string"`
			NewString  string `json:"new_string"`
			ReplaceAll bool   `json:"replace_all"`
		} `json:"edits"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if args.FilePath == "" {
		return "", fmt.Errorf("file_path required")
	}
	if len(args.Edits) == 0 {
		return "", fmt.Errorf("edits required")
	}

	data, err := os.ReadFile(args.FilePath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", args.FilePath, err)
	}
	content := string(data)

	for i, e := range args.Edits {
		if e.OldString == "" {
			return "", fmt.Errorf("edit %d: old_string empty", i)
		}
		if !strings.Contains(content, e.OldString) {
			return "", fmt.Errorf("edit %d: old_string not found in %s", i, args.FilePath)
		}
		if e.ReplaceAll {
			content = strings.ReplaceAll(content, e.OldString, e.NewString)
		} else {
			if strings.Count(content, e.OldString) > 1 {
				return "", fmt.Errorf("edit %d: old_string is not unique in %s; use replace_all or expand context", i, args.FilePath)
			}
			content = strings.Replace(content, e.OldString, e.NewString, 1)
		}
	}

	info, err := os.Stat(args.FilePath)
	mode := os.FileMode(0o644)
	if err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.WriteFile(args.FilePath, []byte(content), mode); err != nil {
		return "", fmt.Errorf("write %s: %w", args.FilePath, err)
	}

	return fmt.Sprintf("applied %d edits to %s", len(args.Edits), args.FilePath), nil
}

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"claudecode/internal/core"
)

type notebookEditTool struct{}

func NewNotebookEdit() core.Tool { return &notebookEditTool{} }

func (t *notebookEditTool) Name() string { return "NotebookEdit" }

func (t *notebookEditTool) Description() string {
	return "Modify a cell in a Jupyter notebook. Find the cell by id or zero-indexed cell_number, then replace, insert, or delete."
}

func (t *notebookEditTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"notebook_path": {"type": "string"},
			"cell_id": {"type": "string"},
			"cell_number": {"type": "integer"},
			"new_source": {"type": "string"},
			"edit_mode": {"type": "string", "enum": ["replace", "insert", "delete"]},
			"cell_type": {"type": "string", "enum": ["code", "markdown"]}
		},
		"required": ["notebook_path", "new_source"],
		"additionalProperties": false
	}`)
}

func (t *notebookEditTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		NotebookPath string  `json:"notebook_path"`
		CellID       string  `json:"cell_id"`
		CellNumber   *int    `json:"cell_number"`
		NewSource    string  `json:"new_source"`
		EditMode     string  `json:"edit_mode"`
		CellType     string  `json:"cell_type"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if args.NotebookPath == "" {
		return "", fmt.Errorf("notebook_path required")
	}
	mode := args.EditMode
	if mode == "" {
		mode = "replace"
	}

	data, err := os.ReadFile(args.NotebookPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", args.NotebookPath, err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse notebook: %w", err)
	}
	var cells []map[string]json.RawMessage
	if raw, ok := doc["cells"]; ok {
		if err := json.Unmarshal(raw, &cells); err != nil {
			return "", fmt.Errorf("parse cells: %w", err)
		}
	}

	idx := -1
	if args.CellID != "" {
		for i, c := range cells {
			if raw, ok := c["id"]; ok {
				var id string
				if json.Unmarshal(raw, &id) == nil && id == args.CellID {
					idx = i
					break
				}
			}
		}
		if idx < 0 && mode != "insert" {
			return "", fmt.Errorf("cell with id %q not found", args.CellID)
		}
	} else if args.CellNumber != nil {
		idx = *args.CellNumber
	}

	cellType := args.CellType
	if cellType == "" {
		cellType = "code"
	}

	switch mode {
	case "replace":
		if idx < 0 || idx >= len(cells) {
			return "", fmt.Errorf("cell index %d out of range", idx)
		}
		srcRaw, _ := json.Marshal(args.NewSource)
		cells[idx]["source"] = srcRaw
		if args.CellType != "" {
			ctRaw, _ := json.Marshal(cellType)
			cells[idx]["cell_type"] = ctRaw
		}
	case "insert":
		newCell := map[string]json.RawMessage{}
		ctRaw, _ := json.Marshal(cellType)
		newCell["cell_type"] = ctRaw
		srcRaw, _ := json.Marshal(args.NewSource)
		newCell["source"] = srcRaw
		newCell["metadata"] = json.RawMessage(`{}`)
		if cellType == "code" {
			newCell["outputs"] = json.RawMessage(`[]`)
			newCell["execution_count"] = json.RawMessage(`null`)
		}
		insertAt := idx
		if insertAt < 0 {
			insertAt = len(cells)
		}
		if insertAt > len(cells) {
			insertAt = len(cells)
		}
		cells = append(cells[:insertAt], append([]map[string]json.RawMessage{newCell}, cells[insertAt:]...)...)
		idx = insertAt
	case "delete":
		if idx < 0 || idx >= len(cells) {
			return "", fmt.Errorf("cell index %d out of range", idx)
		}
		cells = append(cells[:idx], cells[idx+1:]...)
	default:
		return "", fmt.Errorf("unknown edit_mode %q", mode)
	}

	cellsRaw, err := json.Marshal(cells)
	if err != nil {
		return "", fmt.Errorf("marshal cells: %w", err)
	}
	doc["cells"] = cellsRaw
	out, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		return "", fmt.Errorf("marshal notebook: %w", err)
	}
	info, statErr := os.Stat(args.NotebookPath)
	fmode := os.FileMode(0o644)
	if statErr == nil {
		fmode = info.Mode().Perm()
	}
	if err := os.WriteFile(args.NotebookPath, out, fmode); err != nil {
		return "", fmt.Errorf("write %s: %w", args.NotebookPath, err)
	}
	return fmt.Sprintf("modified cell %d in %s", idx, args.NotebookPath), nil
}

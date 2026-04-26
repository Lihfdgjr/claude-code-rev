package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"claudecode/internal/core"
)

type notebookReadTool struct{}

func NewNotebookRead() core.Tool { return &notebookReadTool{} }

func (t *notebookReadTool) Name() string { return "NotebookRead" }

func (t *notebookReadTool) Description() string {
	return "Read a Jupyter notebook (.ipynb) file and return all cells with their sources and outputs."
}

func (t *notebookReadTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"notebook_path": {"type": "string"}
		},
		"required": ["notebook_path"],
		"additionalProperties": false
	}`)
}

type nbCell struct {
	CellType string          `json:"cell_type"`
	ID       string          `json:"id,omitempty"`
	Source   json.RawMessage `json:"source,omitempty"`
	Outputs  []nbOutput      `json:"outputs,omitempty"`
}

type nbOutput struct {
	OutputType string                     `json:"output_type"`
	Text       json.RawMessage            `json:"text,omitempty"`
	Data       map[string]json.RawMessage `json:"data,omitempty"`
	Name       string                     `json:"name,omitempty"`
	Ename      string                     `json:"ename,omitempty"`
	Evalue     string                     `json:"evalue,omitempty"`
	Traceback  []string                   `json:"traceback,omitempty"`
}

type nbDoc struct {
	Cells []nbCell `json:"cells"`
}

func sourceToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return strings.Join(arr, "")
	}
	return string(raw)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...[truncated]"
}

func (t *notebookReadTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		NotebookPath string `json:"notebook_path"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if args.NotebookPath == "" {
		return "", fmt.Errorf("notebook_path required")
	}
	data, err := os.ReadFile(args.NotebookPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", args.NotebookPath, err)
	}
	var doc nbDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse notebook: %w", err)
	}

	var b strings.Builder
	for i, c := range doc.Cells {
		fmt.Fprintf(&b, "### Cell %d (type=%s)\n", i, c.CellType)
		fmt.Fprintf(&b, "%s\n", sourceToString(c.Source))
		for j, o := range c.Outputs {
			fmt.Fprintf(&b, "--- output %d (%s) ---\n", j, o.OutputType)
			switch o.OutputType {
			case "stream":
				fmt.Fprintf(&b, "%s\n", truncate(sourceToString(o.Text), 1000))
			case "error":
				fmt.Fprintf(&b, "%s: %s\n", o.Ename, o.Evalue)
				if len(o.Traceback) > 0 {
					fmt.Fprintf(&b, "%s\n", truncate(strings.Join(o.Traceback, "\n"), 1000))
				}
			default:
				if txt, ok := o.Data["text/plain"]; ok {
					fmt.Fprintf(&b, "%s\n", truncate(sourceToString(txt), 1000))
				} else {
					for mime := range o.Data {
						fmt.Fprintf(&b, "[%s]\n", mime)
					}
				}
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

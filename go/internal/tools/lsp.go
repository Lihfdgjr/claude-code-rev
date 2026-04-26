package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"claudecode/internal/core"
	"claudecode/internal/lsp"
)

type lspPosArgs struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

func parseLSPPos(input json.RawMessage) (lspPosArgs, error) {
	var a lspPosArgs
	if err := json.Unmarshal(input, &a); err != nil {
		return a, fmt.Errorf("invalid input: %w", err)
	}
	if a.File == "" {
		return a, fmt.Errorf("file required")
	}
	return a, nil
}

func lspPosSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file": {"type": "string"},
			"line": {"type": "integer"},
			"column": {"type": "integer"}
		},
		"required": ["file", "line", "column"],
		"additionalProperties": false
	}`)
}

func lspFileSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file": {"type": "string"}
		},
		"required": ["file"],
		"additionalProperties": false
	}`)
}

type lspFileArgs struct {
	File string `json:"file"`
}

func parseLSPFile(input json.RawMessage) (lspFileArgs, error) {
	var a lspFileArgs
	if err := json.Unmarshal(input, &a); err != nil {
		return a, fmt.Errorf("invalid input: %w", err)
	}
	if a.File == "" {
		return a, fmt.Errorf("file required")
	}
	return a, nil
}

var (
	defaultLSPMgrOnce sync.Once
	defaultLSPMgr     *lsp.Manager
)

func lspManager() *lsp.Manager {
	defaultLSPMgrOnce.Do(func() {
		defaultLSPMgr = lsp.DefaultManager()
	})
	return defaultLSPMgr
}

func clientForFile(ctx context.Context, file string) (lsp.Client, error) {
	c, err := lspManager().ForFile(ctx, file)
	if err != nil {
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(file)), ".")
		return nil, fmt.Errorf("%w (extension %q; supported: go, py, ts/tsx/js/jsx, rs, c/cc/cpp/h/hpp, java, rb)", err, ext)
	}
	return c, nil
}

// --- Definition --------------------------------------------------------------

type lspDefinitionTool struct{}

func NewLSPDefinition() core.Tool {
	lspManager()
	return &lspDefinitionTool{}
}

func (t *lspDefinitionTool) Name() string { return "LSPDefinition" }

func (t *lspDefinitionTool) Description() string {
	return "Resolve the definition for a symbol at file:line:column via the LSP backend, routing to the matching language server by extension."
}

func (t *lspDefinitionTool) Schema() json.RawMessage { return lspPosSchema() }

func (t *lspDefinitionTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	a, err := parseLSPPos(input)
	if err != nil {
		return "", err
	}
	c, err := clientForFile(ctx, a.File)
	if err != nil {
		return "", err
	}
	return c.Definition(ctx, a.File, a.Line, a.Column)
}

// --- Hover -------------------------------------------------------------------

type lspHoverTool struct{}

func NewLSPHover() core.Tool {
	lspManager()
	return &lspHoverTool{}
}

func (t *lspHoverTool) Name() string { return "LSPHover" }

func (t *lspHoverTool) Description() string {
	return "Fetch hover documentation for a symbol at file:line:column via the LSP backend, routing to the matching language server by extension."
}

func (t *lspHoverTool) Schema() json.RawMessage { return lspPosSchema() }

func (t *lspHoverTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	a, err := parseLSPPos(input)
	if err != nil {
		return "", err
	}
	c, err := clientForFile(ctx, a.File)
	if err != nil {
		return "", err
	}
	return c.Hover(ctx, a.File, a.Line, a.Column)
}

// --- References --------------------------------------------------------------

type lspReferencesTool struct{}

func NewLSPReferences() core.Tool {
	lspManager()
	return &lspReferencesTool{}
}

func (t *lspReferencesTool) Name() string { return "LSPReferences" }

func (t *lspReferencesTool) Description() string {
	return "List references to the symbol at file:line:column via the LSP backend, routing to the matching language server by extension."
}

func (t *lspReferencesTool) Schema() json.RawMessage { return lspPosSchema() }

func (t *lspReferencesTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	a, err := parseLSPPos(input)
	if err != nil {
		return "", err
	}
	c, err := clientForFile(ctx, a.File)
	if err != nil {
		return "", err
	}
	refs, err := c.References(ctx, a.File, a.Line, a.Column)
	if err != nil {
		return "", err
	}
	return strings.Join(refs, "\n"), nil
}

// --- Symbols -----------------------------------------------------------------

type lspSymbolsTool struct{}

func NewLSPSymbols() core.Tool {
	lspManager()
	return &lspSymbolsTool{}
}

func (t *lspSymbolsTool) Name() string { return "LSPSymbols" }

func (t *lspSymbolsTool) Description() string {
	return "List document symbols for a file via the LSP backend, routing to the matching language server by extension."
}

func (t *lspSymbolsTool) Schema() json.RawMessage { return lspFileSchema() }

func (t *lspSymbolsTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	a, err := parseLSPFile(input)
	if err != nil {
		return "", err
	}
	c, err := clientForFile(ctx, a.File)
	if err != nil {
		return "", err
	}
	syms, err := c.Symbols(ctx, a.File)
	if err != nil {
		return "", err
	}
	return strings.Join(syms, "\n"), nil
}

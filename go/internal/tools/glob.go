package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"claudecode/internal/core"
)

const globMaxResults = 250

type globTool struct{}

type globInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

func NewGlob() core.Tool { return &globTool{} }

func (globTool) Name() string { return "Glob" }

func (globTool) Description() string {
	return "Find files matching a glob pattern. Supports '**' for recursive matching. Results sorted by modification time, newest first."
}

func (globTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "pattern": {"type": "string", "description": "Glob pattern (supports '**' for recursion)"},
    "path": {"type": "string", "description": "Base directory (default cwd)"}
  },
  "required": ["pattern"],
  "additionalProperties": false
}`)
}

type globEntry struct {
	path  string
	mtime time.Time
}

func (globTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in globInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	base := in.Path
	if base == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		base = cwd
	}

	pattern := filepath.ToSlash(in.Pattern)
	var entries []globEntry

	if strings.Contains(pattern, "**") {
		err := filepath.Walk(base, func(p string, fi os.FileInfo, werr error) error {
			if werr != nil {
				return nil
			}
			if fi.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(base, p)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if matchDoubleStar(pattern, rel) {
				entries = append(entries, globEntry{path: p, mtime: fi.ModTime()})
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	} else {
		full := pattern
		if !filepath.IsAbs(full) {
			full = filepath.Join(base, pattern)
		}
		matches, err := filepath.Glob(full)
		if err != nil {
			return "", err
		}
		for _, m := range matches {
			fi, err := os.Stat(m)
			if err != nil {
				continue
			}
			if fi.IsDir() {
				continue
			}
			entries = append(entries, globEntry{path: m, mtime: fi.ModTime()})
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].mtime.After(entries[j].mtime) })
	if len(entries) > globMaxResults {
		entries = entries[:globMaxResults]
	}

	var b strings.Builder
	for _, e := range entries {
		b.WriteString(e.path)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// matchDoubleStar matches a slash-separated path against a pattern that may
// include '**' segments. '**' matches zero or more path components.
func matchDoubleStar(pattern, name string) bool {
	pParts := strings.Split(pattern, "/")
	nParts := strings.Split(name, "/")
	return matchParts(pParts, nParts)
}

func matchParts(pat, name []string) bool {
	for len(pat) > 0 {
		p := pat[0]
		if p == "**" {
			// collapse consecutive **
			for len(pat) > 1 && pat[1] == "**" {
				pat = pat[1:]
			}
			rest := pat[1:]
			if len(rest) == 0 {
				return true
			}
			for i := 0; i <= len(name); i++ {
				if matchParts(rest, name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		ok, err := filepath.Match(p, name[0])
		if err != nil || !ok {
			return false
		}
		pat = pat[1:]
		name = name[1:]
	}
	return len(name) == 0
}

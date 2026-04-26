package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeGrepTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"a.txt":     "hello world\nfoo bar\nHELLO again\n",
		"b.go":      "package main\n// hello there\n",
		"c.md":      "no matches in here\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func TestGrepFindsMatches(t *testing.T) {
	dir := makeGrepTree(t)
	tool := NewGrep()
	in, _ := json.Marshal(map[string]any{
		"pattern": "hello",
		"path":    dir,
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "a.txt") || !strings.Contains(out, "b.go") {
		t.Errorf("expected files_with_matches to include a.txt and b.go, got: %q", out)
	}
	if strings.Contains(out, "c.md") {
		t.Errorf("c.md should not match: %q", out)
	}
}

func TestGrepCaseInsensitive(t *testing.T) {
	dir := makeGrepTree(t)
	tool := NewGrep()
	in, _ := json.Marshal(map[string]any{
		"pattern":          "HELLO",
		"path":             dir,
		"case_insensitive": true,
		"output_mode":      "content",
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// "hello world" (lower), "HELLO again" (upper), "// hello there" should all match.
	if !strings.Contains(out, "hello world") {
		t.Errorf("missing 'hello world' in output: %q", out)
	}
	if !strings.Contains(out, "HELLO again") {
		t.Errorf("missing 'HELLO again' in output: %q", out)
	}
}

func TestGrepCaseSensitiveDoesNotMatchOpposite(t *testing.T) {
	dir := makeGrepTree(t)
	tool := NewGrep()
	in, _ := json.Marshal(map[string]any{
		"pattern":     "HELLO",
		"path":        dir,
		"output_mode": "content",
	})
	out, _ := tool.Run(context.Background(), in)
	if strings.Contains(out, "hello world") {
		t.Errorf("case-sensitive should not match 'hello world': %q", out)
	}
}

func TestGrepOutputModeCount(t *testing.T) {
	dir := makeGrepTree(t)
	tool := NewGrep()
	in, _ := json.Marshal(map[string]any{
		"pattern":     "hello",
		"path":        dir,
		"output_mode": "count",
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// a.txt: 1 (hello world), b.go: 1 (// hello there). Each line "path:N".
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 result lines, got %d: %q", len(lines), out)
	}
	for _, ln := range lines {
		// Path should end with :1
		if !strings.HasSuffix(ln, ":1") {
			t.Errorf("expected count line ending with ':1', got %q", ln)
		}
	}
}

func TestGrepRequiresPattern(t *testing.T) {
	tool := NewGrep()
	in, _ := json.Marshal(map[string]any{})
	_, err := tool.Run(context.Background(), in)
	if err == nil {
		t.Error("expected error for missing pattern")
	}
}

func TestGrepInvalidPattern(t *testing.T) {
	tool := NewGrep()
	in, _ := json.Marshal(map[string]any{"pattern": "[unclosed"})
	_, err := tool.Run(context.Background(), in)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

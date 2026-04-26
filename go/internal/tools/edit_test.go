package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestEditMultipleMatchesWithoutReplaceAllErrors(t *testing.T) {
	p := writeTempFile(t, "foo bar foo baz")
	tool := NewEdit()
	in, _ := json.Marshal(map[string]any{
		"file_path":  p,
		"old_string": "foo",
		"new_string": "qux",
	})
	_, err := tool.Run(context.Background(), in)
	if err == nil {
		t.Fatal("expected error for multiple matches without replace_all")
	}
	if !strings.Contains(err.Error(), "not unique") {
		t.Errorf("expected 'not unique' in error, got %v", err)
	}
}

func TestEditNoMatchErrors(t *testing.T) {
	p := writeTempFile(t, "foo bar")
	tool := NewEdit()
	in, _ := json.Marshal(map[string]any{
		"file_path":  p,
		"old_string": "missing",
		"new_string": "x",
	})
	_, err := tool.Run(context.Background(), in)
	if err == nil {
		t.Fatal("expected error for no match")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got %v", err)
	}
}

func TestEditSingleReplaceSuccess(t *testing.T) {
	p := writeTempFile(t, "alpha beta gamma")
	tool := NewEdit()
	in, _ := json.Marshal(map[string]any{
		"file_path":  p,
		"old_string": "beta",
		"new_string": "BETA",
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "1 replacements") {
		t.Errorf("expected '1 replacements' in output, got %q", out)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "alpha BETA gamma" {
		t.Errorf("file content = %q", string(got))
	}
}

func TestEditReplaceAllSuccess(t *testing.T) {
	p := writeTempFile(t, "a b a b a")
	tool := NewEdit()
	in, _ := json.Marshal(map[string]any{
		"file_path":   p,
		"old_string":  "a",
		"new_string":  "A",
		"replace_all": true,
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "3 replacements") {
		t.Errorf("expected '3 replacements' in output, got %q", out)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "A b A b A" {
		t.Errorf("file content = %q", string(got))
	}
}

func TestEditRequiresAbsolutePath(t *testing.T) {
	tool := NewEdit()
	in, _ := json.Marshal(map[string]any{
		"file_path":  "rel.txt",
		"old_string": "x",
		"new_string": "y",
	})
	_, err := tool.Run(context.Background(), in)
	if err == nil {
		t.Error("expected error for relative path")
	}
}

func TestEditRejectsIdenticalStrings(t *testing.T) {
	p := writeTempFile(t, "hi")
	tool := NewEdit()
	in, _ := json.Marshal(map[string]any{
		"file_path":  p,
		"old_string": "x",
		"new_string": "x",
	})
	_, err := tool.Run(context.Background(), in)
	if err == nil {
		t.Error("expected error for identical strings")
	}
}

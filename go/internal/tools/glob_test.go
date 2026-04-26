package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeGlobTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := []string{
		"main.go",
		"util.go",
		"README.md",
		"sub/a.txt",
		"sub/b.txt",
		"sub/deeper/c.txt",
		"sub/deeper/d.go",
	}
	for _, f := range files {
		full := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return dir
}

func TestGlobMatchesStarInBase(t *testing.T) {
	dir := makeGlobTree(t)
	tool := NewGlob()
	in, _ := json.Marshal(map[string]any{
		"pattern": "*.go",
		"path":    dir,
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "main.go") || !strings.Contains(out, "util.go") {
		t.Errorf("expected main.go and util.go: %q", out)
	}
	// Sub-dir Go files should not appear without **.
	if strings.Contains(out, "deeper") {
		t.Errorf("non-recursive *.go should not match nested files: %q", out)
	}
	if strings.Contains(out, "README.md") {
		t.Errorf("*.go should not match README.md: %q", out)
	}
}

func TestGlobDoubleStarRecursiveTxt(t *testing.T) {
	dir := makeGlobTree(t)
	tool := NewGlob()
	in, _ := json.Marshal(map[string]any{
		"pattern": "**/*.txt",
		"path":    dir,
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, want := range []string{"a.txt", "b.txt", "c.txt"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in output: %q", want, out)
		}
	}
	if strings.Contains(out, "main.go") {
		t.Errorf("**/*.txt should not match Go files: %q", out)
	}
}

func TestGlobRequiresPattern(t *testing.T) {
	tool := NewGlob()
	in, _ := json.Marshal(map[string]any{})
	_, err := tool.Run(context.Background(), in)
	if err == nil {
		t.Error("expected error for missing pattern")
	}
}

func TestGlobMatchDoubleStarHelper(t *testing.T) {
	cases := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"**/*.txt", "a.txt", true},
		{"**/*.txt", "sub/b.txt", true},
		{"**/*.txt", "sub/deep/c.txt", true},
		{"**/*.txt", "main.go", false},
		{"sub/**", "sub/a/b/c.txt", true},
		{"sub/**", "other/a.txt", false},
	}
	for _, tc := range cases {
		got := matchDoubleStar(tc.pattern, tc.name)
		if got != tc.want {
			t.Errorf("matchDoubleStar(%q, %q) = %v, want %v", tc.pattern, tc.name, got, tc.want)
		}
	}
}

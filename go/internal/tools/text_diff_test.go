package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTextDiffShowsChange(t *testing.T) {
	tool := NewTextDiff()
	in, _ := json.Marshal(map[string]any{
		"a": "hello\nworld",
		"b": "hello\ngo",
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "@@") {
		t.Errorf("expected hunk header in diff, got %q", out)
	}
	if !strings.Contains(out, "-world") {
		t.Errorf("expected '-world' in diff, got %q", out)
	}
	if !strings.Contains(out, "+go") {
		t.Errorf("expected '+go' in diff, got %q", out)
	}
	if !strings.Contains(out, " hello") {
		t.Errorf("expected ' hello' context in diff, got %q", out)
	}
	if !strings.HasPrefix(out, "--- a\n+++ b\n") {
		t.Errorf("expected unified diff header, got %q", out)
	}
}

func TestTextDiffEmptyWhenIdentical(t *testing.T) {
	tool := NewTextDiff()
	in, _ := json.Marshal(map[string]any{
		"a": "same\nstuff\n",
		"b": "same\nstuff\n",
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty diff for identical inputs, got %q", out)
	}
}

func TestTextDiffPureAddition(t *testing.T) {
	tool := NewTextDiff()
	in, _ := json.Marshal(map[string]any{
		"a": "",
		"b": "new\n",
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "+new") {
		t.Errorf("expected '+new' in diff, got %q", out)
	}
}

func TestTextDiffPureDeletion(t *testing.T) {
	tool := NewTextDiff()
	in, _ := json.Marshal(map[string]any{
		"a": "removed\n",
		"b": "",
	})
	out, err := tool.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "-removed") {
		t.Errorf("expected '-removed' in diff, got %q", out)
	}
}

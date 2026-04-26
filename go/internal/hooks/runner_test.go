package hooks

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestRunnerEmptyConfigNoOp(t *testing.T) {
	r := New(nil)
	d, err := r.Run(context.Background(), Event{Name: PreToolUse, ToolName: "Bash"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if d.Block {
		t.Errorf("expected non-blocking decision, got %+v", d)
	}
	if d.Reason != "" {
		t.Errorf("expected empty reason, got %q", d.Reason)
	}
}

func TestRunnerExitZeroContinues(t *testing.T) {
	cmd := "exit 0"
	if runtime.GOOS == "windows" {
		cmd = "exit /b 0"
	}
	r := New(Config{
		PreToolUse: []HookSpec{{Matcher: "Bash", Type: "command", Command: cmd}},
	})
	d, err := r.Run(context.Background(), Event{Name: PreToolUse, ToolName: "Bash"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if d.Block {
		t.Errorf("exit 0 hook should not block, got %+v", d)
	}
}

func TestRunnerExitNonZeroBlocksWithStderr(t *testing.T) {
	var cmd string
	if runtime.GOOS == "windows" {
		// Write to stderr then exit non-zero on cmd.exe.
		cmd = "echo nope 1>&2 & exit /b 7"
	} else {
		cmd = "echo nope 1>&2; exit 7"
	}
	r := New(Config{
		PreToolUse: []HookSpec{{Matcher: "Bash", Type: "command", Command: cmd}},
	})
	d, err := r.Run(context.Background(), Event{Name: PreToolUse, ToolName: "Bash"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !d.Block {
		t.Errorf("expected blocking decision, got %+v", d)
	}
	if !strings.Contains(d.Reason, "nope") {
		t.Errorf("expected 'nope' in stderr reason, got %q", d.Reason)
	}
}

func TestRunnerMatcherSkipsNonMatching(t *testing.T) {
	// Hook only matches Bash but we send Edit event - should not run.
	cmd := "exit 1"
	if runtime.GOOS == "windows" {
		cmd = "exit /b 1"
	}
	r := New(Config{
		PreToolUse: []HookSpec{{Matcher: "^Bash$", Type: "command", Command: cmd}},
	})
	d, err := r.Run(context.Background(), Event{Name: PreToolUse, ToolName: "Edit"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if d.Block {
		t.Errorf("expected non-blocking decision when matcher excludes tool, got %+v", d)
	}
}

func TestRunnerNoSpecsForEvent(t *testing.T) {
	r := New(Config{
		PreToolUse: []HookSpec{{Type: "command", Command: "exit 1"}},
	})
	d, err := r.Run(context.Background(), Event{Name: PostToolUse, ToolName: "Bash"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if d.Block {
		t.Errorf("PostToolUse with no PostToolUse hooks should not block, got %+v", d)
	}
}

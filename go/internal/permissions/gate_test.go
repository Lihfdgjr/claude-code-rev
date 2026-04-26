package permissions

import (
	"context"
	"testing"

	"claudecode/internal/core"
)

func TestGateDefaultModeAllow(t *testing.T) {
	g := New(Config{})
	d, _ := g.Check(context.Background(), core.PermissionRequest{Tool: "AnyTool"})
	if d != core.PermissionAllow {
		t.Errorf("default mode decision = %v, want allow", d)
	}
}

func TestGateModeDeny(t *testing.T) {
	g := New(Config{Mode: "deny"})
	d, _ := g.Check(context.Background(), core.PermissionRequest{Tool: "AnyTool"})
	if d != core.PermissionDeny {
		t.Errorf("mode deny decision = %v, want deny", d)
	}
}

func TestGateModeAsk(t *testing.T) {
	g := New(Config{Mode: "ask"})
	d, _ := g.Check(context.Background(), core.PermissionRequest{Tool: "AnyTool"})
	if d != core.PermissionAsk {
		t.Errorf("mode ask decision = %v, want ask", d)
	}
}

func TestGateExplicitAllowList(t *testing.T) {
	g := New(Config{Mode: "ask", AllowedTools: []string{"GoodTool"}})
	d, reason := g.Check(context.Background(), core.PermissionRequest{Tool: "GoodTool"})
	if d != core.PermissionAllow {
		t.Errorf("explicit allow decision = %v, want allow (reason: %s)", d, reason)
	}
	if reason == "" {
		t.Error("expected reason text")
	}
}

func TestGateExplicitDenyList(t *testing.T) {
	g := New(Config{Mode: "allow", DeniedTools: []string{"BadTool"}})
	d, _ := g.Check(context.Background(), core.PermissionRequest{Tool: "BadTool"})
	if d != core.PermissionDeny {
		t.Errorf("explicit deny decision = %v, want deny", d)
	}
}

func TestGateDenyOverridesAllow(t *testing.T) {
	g := New(Config{
		Mode:         "ask",
		AllowedTools: []string{"BadTool"},
		DeniedTools:  []string{"BadTool"},
	})
	d, _ := g.Check(context.Background(), core.PermissionRequest{Tool: "BadTool"})
	if d != core.PermissionDeny {
		t.Errorf("deny should override allow, got %v", d)
	}
}

func TestGateAllowRuntimeAddsToAllowed(t *testing.T) {
	g := New(Config{Mode: "ask"})
	if d, _ := g.Check(context.Background(), core.PermissionRequest{Tool: "Pending"}); d != core.PermissionAsk {
		t.Fatalf("pre-AllowRuntime decision = %v, want ask", d)
	}
	g.AllowRuntime("Pending")
	if d, _ := g.Check(context.Background(), core.PermissionRequest{Tool: "Pending"}); d != core.PermissionAllow {
		t.Errorf("post-AllowRuntime decision = %v, want allow", d)
	}
}

func TestGateAllowRuntimeIgnoresEmpty(t *testing.T) {
	g := New(Config{Mode: "ask"})
	g.AllowRuntime("")
	if d, _ := g.Check(context.Background(), core.PermissionRequest{Tool: ""}); d != core.PermissionAsk {
		t.Errorf("empty AllowRuntime should not register, got %v", d)
	}
}

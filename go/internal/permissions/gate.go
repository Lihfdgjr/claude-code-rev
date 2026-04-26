package permissions

import (
	"context"
	"sync"

	"claudecode/internal/core"
)

type Config struct {
	Mode         string
	AllowedTools []string
	DeniedTools  []string
}

type Gate struct {
	mu      sync.RWMutex
	mode    string
	allowed map[string]struct{}
	denied  map[string]struct{}
}

func New(cfg Config) *Gate {
	g := &Gate{
		mode:    cfg.Mode,
		allowed: make(map[string]struct{}, len(cfg.AllowedTools)),
		denied:  make(map[string]struct{}, len(cfg.DeniedTools)),
	}
	if g.mode == "" {
		g.mode = "allow"
	}
	for _, t := range cfg.AllowedTools {
		g.allowed[t] = struct{}{}
	}
	for _, t := range cfg.DeniedTools {
		g.denied[t] = struct{}{}
	}
	return g
}

func (g *Gate) Check(ctx context.Context, req core.PermissionRequest) (core.PermissionDecision, string) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if _, ok := g.denied[req.Tool]; ok {
		return core.PermissionDeny, "explicit deny rule"
	}
	if _, ok := g.allowed[req.Tool]; ok {
		return core.PermissionAllow, "explicit allow rule"
	}
	switch g.mode {
	case "deny":
		return core.PermissionDeny, "default mode deny"
	case "ask":
		return core.PermissionAsk, "default mode ask"
	default:
		return core.PermissionAllow, "default mode allow"
	}
}

func (g *Gate) AllowRuntime(tool string) {
	if tool == "" {
		return
	}
	g.mu.Lock()
	g.allowed[tool] = struct{}{}
	g.mu.Unlock()
}

func (g *Gate) Reconfigure(cfg Config) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.mode = cfg.Mode
	if g.mode == "" {
		g.mode = "allow"
	}
	g.allowed = make(map[string]struct{}, len(cfg.AllowedTools))
	g.denied = make(map[string]struct{}, len(cfg.DeniedTools))
	for _, t := range cfg.AllowedTools {
		g.allowed[t] = struct{}{}
	}
	for _, t := range cfg.DeniedTools {
		g.denied[t] = struct{}{}
	}
}

func (g *Gate) SetAllowed(tools []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.allowed = make(map[string]struct{}, len(tools))
	for _, t := range tools {
		g.allowed[t] = struct{}{}
	}
}

func (g *Gate) SetDenied(tools []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.denied = make(map[string]struct{}, len(tools))
	for _, t := range tools {
		g.denied[t] = struct{}{}
	}
}

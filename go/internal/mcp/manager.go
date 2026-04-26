package mcp

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"claudecode/internal/core"
)

// ActiveManager is the package-level handle set by main.go on startup so
// commands can reach the running Manager without import cycles.
var ActiveManager *Manager

// SetActive registers the running manager as the package-level handle.
func SetActive(m *Manager) { ActiveManager = m }

// Manager owns a set of MCP clients keyed by server name and aggregates their tools.
type Manager struct {
	mu      sync.Mutex
	clients map[string]clientLike
	tools   []core.Tool
}

// NewManager returns an empty manager.
func NewManager() *Manager {
	return &Manager{clients: make(map[string]clientLike)}
}

// newClientForConfig picks the appropriate transport for cfg.
func newClientForConfig(cfg Config) clientLike {
	transport := strings.ToLower(strings.TrimSpace(cfg.Transport))
	if transport == "sse" || (transport == "" && strings.TrimSpace(cfg.URL) != "") {
		return NewSSE(SSEConfig{URL: cfg.URL, Headers: cfg.Env})
	}
	return New(cfg)
}

// Start brings up every configured server in parallel. A failed server is
// logged to stderr and skipped; the others continue to come up.
func (m *Manager) Start(ctx context.Context, servers map[string]Config) error {
	if len(servers) == 0 {
		return nil
	}

	type result struct {
		name   string
		client clientLike
		tools  []MCPTool
		err    error
	}
	results := make(chan result, len(servers))

	var wg sync.WaitGroup
	for name, cfg := range servers {
		cfg := cfg
		if cfg.Name == "" {
			cfg.Name = name
		}
		wg.Add(1)
		go func(serverName string, c Config) {
			defer wg.Done()
			cli := newClientForConfig(c)
			if err := cli.Start(ctx); err != nil {
				results <- result{name: serverName, err: err}
				return
			}
			tools, err := cli.ListTools()
			if err != nil {
				_ = cli.Stop()
				results <- result{name: serverName, err: err}
				return
			}
			results <- result{name: serverName, client: cli, tools: tools}
		}(name, cfg)
	}
	wg.Wait()
	close(results)

	m.mu.Lock()
	defer m.mu.Unlock()
	for r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "mcp: server %q failed: %v\n", r.name, r.err)
			continue
		}
		m.clients[r.name] = r.client
		for _, t := range r.tools {
			prefixed := fmt.Sprintf("mcp__%s__%s", r.name, t.Name)
			m.tools = append(m.tools, newTool(r.client, prefixed, t))
		}
	}
	return nil
}

// Stop shuts down every active client. The first error is returned.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var firstErr error
	for name, c := range m.clients {
		if err := c.Stop(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("mcp: stop %s: %w", name, err)
		}
		delete(m.clients, name)
	}
	m.tools = nil
	return firstErr
}

// Tools returns a snapshot of all discovered MCP tools as core.Tool wrappers.
func (m *Manager) Tools() []core.Tool {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]core.Tool, len(m.tools))
	copy(out, m.tools)
	return out
}

// Names returns the sorted list of currently active server names.
func (m *Manager) Names() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.clients))
	for n := range m.clients {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Restart stops and removes the named client (and its cached tools), then
// starts a fresh one from cfg and re-discovers its tools.
func (m *Manager) Restart(ctx context.Context, name string, cfg Config) error {
	if name == "" {
		return fmt.Errorf("mcp: empty server name")
	}
	if cfg.Name == "" {
		cfg.Name = name
	}

	m.mu.Lock()
	if old, ok := m.clients[name]; ok {
		_ = old.Stop()
		delete(m.clients, name)
	}
	prefix := fmt.Sprintf("mcp__%s__", name)
	filtered := m.tools[:0]
	for _, t := range m.tools {
		if !strings.HasPrefix(t.Name(), prefix) {
			filtered = append(filtered, t)
		}
	}
	m.tools = filtered
	m.mu.Unlock()

	cli := newClientForConfig(cfg)
	if err := cli.Start(ctx); err != nil {
		return fmt.Errorf("mcp: start %s: %w", name, err)
	}
	tools, err := cli.ListTools()
	if err != nil {
		_ = cli.Stop()
		return fmt.Errorf("mcp: list tools %s: %w", name, err)
	}

	m.mu.Lock()
	m.clients[name] = cli
	for _, t := range tools {
		prefixed := fmt.Sprintf("mcp__%s__%s", name, t.Name)
		m.tools = append(m.tools, newTool(cli, prefixed, t))
	}
	m.mu.Unlock()
	return nil
}

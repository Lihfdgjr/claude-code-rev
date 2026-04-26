package lsp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// ServerSpec describes a language server and the file extensions it serves.
// Languages are bare extensions (no leading dot), e.g. "go", "ts".
type ServerSpec struct {
	Languages []string
	Command   string
	Args      []string
}

// DefaultServers is the built-in routing table from extension to language server.
var DefaultServers = []ServerSpec{
	{Languages: []string{"go"}, Command: "gopls"},
	{Languages: []string{"py"}, Command: "pyright-langserver", Args: []string{"--stdio"}},
	{Languages: []string{"ts", "tsx", "js", "jsx"}, Command: "typescript-language-server", Args: []string{"--stdio"}},
	{Languages: []string{"rs"}, Command: "rust-analyzer"},
	{Languages: []string{"c", "cc", "cpp", "h", "hpp"}, Command: "clangd"},
	{Languages: []string{"java"}, Command: "jdtls"},
	{Languages: []string{"rb"}, Command: "solargraph", Args: []string{"stdio"}},
}

// Manager owns a pool of lazily-started LSP clients keyed by command name.
type Manager struct {
	mu      sync.Mutex
	clients map[string]Client
	specs   []ServerSpec
}

// NewManager builds a Manager that routes files using the given specs.
func NewManager(specs []ServerSpec) *Manager {
	return &Manager{
		clients: make(map[string]Client),
		specs:   specs,
	}
}

var (
	defaultManagerOnce sync.Once
	defaultManager     *Manager
)

// DefaultManager returns the process-wide Manager backed by DefaultServers.
func DefaultManager() *Manager {
	defaultManagerOnce.Do(func() {
		defaultManager = NewManager(DefaultServers)
	})
	return defaultManager
}

// ForFile resolves the right Client for the given file path, lazily starting
// the underlying language server on first use.
func (m *Manager) ForFile(ctx context.Context, file string) (Client, error) {
	if file == "" {
		return nil, fmt.Errorf("lsp: file required")
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(file)), ".")
	if ext == "" {
		return nil, fmt.Errorf("lsp: file %q has no extension", file)
	}
	spec, ok := m.findSpec(ext)
	if !ok {
		return nil, fmt.Errorf("lsp: no server registered for extension %q", ext)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clients[spec.Command]; ok {
		if started, canCheck := c.(interface{ Started() bool }); !canCheck || started.Started() {
			return c, nil
		}
		if err := c.Start(ctx, spec.Command, spec.Args); err != nil {
			return nil, fmt.Errorf("lsp: start %s: %w", spec.Command, err)
		}
		return c, nil
	}
	c := New()
	if err := c.Start(ctx, spec.Command, spec.Args); err != nil {
		return nil, fmt.Errorf("lsp: start %s: %w", spec.Command, err)
	}
	m.clients[spec.Command] = c
	return c, nil
}

// Stop shuts down every started client, returning the first error if any.
func (m *Manager) Stop() {
	m.mu.Lock()
	clients := make([]Client, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	m.clients = make(map[string]Client)
	m.mu.Unlock()
	for _, c := range clients {
		_ = c.Stop()
	}
}

func (m *Manager) findSpec(ext string) (ServerSpec, bool) {
	for _, s := range m.specs {
		for _, l := range s.Languages {
			if strings.EqualFold(l, ext) {
				return s, true
			}
		}
	}
	return ServerSpec{}, false
}

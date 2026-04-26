package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/mcp"
)

type mcpCmd struct{}

func NewMCP() core.Command { return &mcpCmd{} }

func (mcpCmd) Name() string     { return "mcp" }
func (mcpCmd) Synopsis() string { return "Manage MCP servers" }

type mcpServerEntry struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func (mcpCmd) Run(ctx context.Context, args string, sess core.Session) error {
	fields := strings.Fields(strings.TrimSpace(args))
	if len(fields) == 0 {
		return mcpList(sess)
	}
	switch fields[0] {
	case "add":
		if len(fields) < 3 {
			sess.Notify(core.NotifyError, "usage: /mcp add <name> <command> [args...]")
			return nil
		}
		return mcpAdd(sess, fields[1], fields[2], fields[3:])
	case "remove", "rm":
		if len(fields) < 2 {
			sess.Notify(core.NotifyError, "usage: /mcp remove <name>")
			return nil
		}
		return mcpRemove(sess, fields[1])
	case "restart":
		var target string
		if len(fields) >= 2 {
			target = fields[1]
		}
		return mcpRestart(ctx, sess, target)
	default:
		sess.Notify(core.NotifyError, fmt.Sprintf("unknown subcommand: %s", fields[0]))
		return nil
	}
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func readSettings(path string) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func writeSettingsAtomic(path string, settings map[string]json.RawMessage) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func mcpList(sess core.Session) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	settings, err := readSettings(path)
	if err != nil {
		return err
	}
	raw, ok := settings["mcpServers"]
	if !ok || len(raw) == 0 {
		sess.Notify(core.NotifyInfo, "No MCP servers configured in "+path)
		return nil
	}
	var servers map[string]mcpServerEntry
	if err := json.Unmarshal(raw, &servers); err != nil {
		return err
	}
	if len(servers) == 0 {
		sess.Notify(core.NotifyInfo, "No MCP servers configured in "+path)
		return nil
	}
	names := make([]string, 0, len(servers))
	for n := range servers {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("MCP servers:\n")
	for _, n := range names {
		s := servers[n]
		if s.URL != "" {
			fmt.Fprintf(&b, "  %s url=%s\n", n, s.URL)
			continue
		}
		if len(s.Args) > 0 {
			fmt.Fprintf(&b, "  %s %s %s\n", n, s.Command, strings.Join(s.Args, " "))
		} else {
			fmt.Fprintf(&b, "  %s %s\n", n, s.Command)
		}
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}

func mcpAdd(sess core.Session, name, command string, args []string) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	settings, err := readSettings(path)
	if err != nil {
		return err
	}
	servers := map[string]mcpServerEntry{}
	if raw, ok := settings["mcpServers"]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return err
		}
	}
	servers[name] = mcpServerEntry{Command: command, Args: args}
	updated, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	settings["mcpServers"] = updated
	if err := writeSettingsAtomic(path, settings); err != nil {
		return err
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Added MCP server %q to %s", name, path))
	return nil
}

// loadServerConfigs reads settings.json and returns the configured MCP servers
// as mcp.Config values keyed by name.
func loadServerConfigs() (map[string]mcp.Config, error) {
	path, err := settingsPath()
	if err != nil {
		return nil, err
	}
	settings, err := readSettings(path)
	if err != nil {
		return nil, err
	}
	raw, ok := settings["mcpServers"]
	if !ok || len(raw) == 0 {
		return nil, nil
	}
	var entries map[string]mcpServerEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	out := make(map[string]mcp.Config, len(entries))
	for name, e := range entries {
		cfg := mcp.Config{
			Name:    name,
			Command: e.Command,
			Args:    e.Args,
			URL:     e.URL,
			Env:     e.Env,
		}
		if e.URL != "" {
			cfg.Transport = "sse"
		}
		out[name] = cfg
	}
	return out, nil
}

func mcpRestart(ctx context.Context, sess core.Session, name string) error {
	if mcp.ActiveManager == nil {
		sess.Notify(core.NotifyInfo, "MCP manager not available; restart the CLI to reconnect MCP servers.")
		return nil
	}
	configs, err := loadServerConfigs()
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("mcp restart: read settings: %v", err))
		return err
	}

	var targets []string
	if name != "" {
		if _, ok := configs[name]; !ok {
			sess.Notify(core.NotifyError, fmt.Sprintf("mcp restart: server %q not configured", name))
			return nil
		}
		targets = []string{name}
	} else {
		for n := range configs {
			targets = append(targets, n)
		}
		sort.Strings(targets)
	}
	if len(targets) == 0 {
		sess.Notify(core.NotifyInfo, "No MCP servers configured.")
		return nil
	}

	for _, n := range targets {
		if err := mcp.ActiveManager.Restart(ctx, n, configs[n]); err != nil {
			sess.Notify(core.NotifyError, fmt.Sprintf("mcp restart %s: %v", n, err))
			continue
		}
		sess.Notify(core.NotifyInfo, fmt.Sprintf("Restarted MCP server %q", n))
	}
	return nil
}

func mcpRemove(sess core.Session, name string) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	settings, err := readSettings(path)
	if err != nil {
		return err
	}
	raw, ok := settings["mcpServers"]
	if !ok || len(raw) == 0 {
		sess.Notify(core.NotifyInfo, fmt.Sprintf("No MCP server %q to remove.", name))
		return nil
	}
	servers := map[string]mcpServerEntry{}
	if err := json.Unmarshal(raw, &servers); err != nil {
		return err
	}
	if _, ok := servers[name]; !ok {
		sess.Notify(core.NotifyInfo, fmt.Sprintf("No MCP server %q to remove.", name))
		return nil
	}
	delete(servers, name)
	updated, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	settings["mcpServers"] = updated
	if err := writeSettingsAtomic(path, settings); err != nil {
		return err
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Removed MCP server %q from %s", name, path))
	return nil
}

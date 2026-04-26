package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claudecode/internal/core"
)

type ManifestCommand struct {
	Name     string `json:"name"`
	Synopsis string `json:"synopsis"`
	Run      string `json:"run"`
}

type ManifestTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Run         string          `json:"run"`
}

type Manifest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Version     string              `json:"version"`
	Commands    []ManifestCommand   `json:"commands"`
	Tools       []ManifestTool      `json:"tools"`
	Hooks       map[string][]string `json:"hooks"`
}

type Plugin struct {
	Name        string
	Description string
	Version     string
	Path        string
	Manifest    *Manifest
	Commands    []core.Command
	Tools       []core.Tool
}

type Loader struct {
	homeDir    string
	projectDir string
}

func New(homeDir, projectDir string) *Loader {
	return &Loader{homeDir: homeDir, projectDir: projectDir}
}

func (l *Loader) Load() ([]*Plugin, error) {
	var roots []string
	if l.homeDir != "" {
		roots = append(roots, filepath.Join(l.homeDir, ".claude", "plugins"))
	}
	if l.projectDir != "" {
		roots = append(roots, filepath.Join(l.projectDir, ".claude", "plugins"))
	}

	var plugins []*Plugin
	seen := map[string]bool{}

	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Fprintf(os.Stderr, "plugins: read %s: %v\n", root, err)
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(root, e.Name())
			manifestPath := filepath.Join(dir, "plugin.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				if !os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "plugins: read %s: %v\n", manifestPath, err)
				}
				continue
			}
			var m Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				fmt.Fprintf(os.Stderr, "plugins: parse %s: %v\n", manifestPath, err)
				continue
			}
			name := m.Name
			if name == "" {
				name = e.Name()
			}
			if seen[name] {
				continue
			}
			seen[name] = true

			p := &Plugin{
				Name:        name,
				Description: m.Description,
				Version:     m.Version,
				Path:        dir,
				Manifest:    &m,
			}
			for _, mc := range m.Commands {
				p.Commands = append(p.Commands, newPluginCommand(mc, dir))
			}
			for _, mt := range m.Tools {
				p.Tools = append(p.Tools, newPluginTool(mt, dir))
			}
			plugins = append(plugins, p)
		}
	}

	return plugins, nil
}

type pluginCommand struct {
	name     string
	synopsis string
	run      string
	cwd      string
}

func newPluginCommand(mc ManifestCommand, cwd string) core.Command {
	return &pluginCommand{
		name:     mc.Name,
		synopsis: mc.Synopsis,
		run:      mc.Run,
		cwd:      cwd,
	}
}

func (c *pluginCommand) Name() string     { return c.name }
func (c *pluginCommand) Synopsis() string { return c.synopsis }

func (c *pluginCommand) Run(ctx context.Context, args string, sess core.Session) error {
	cmd := strings.ReplaceAll(c.run, "{{args}}", args)
	env := map[string]string{
		"CLAUDE_PLUGIN_DIR": c.cwd,
		"CLAUDE_PLUGIN":     c.name,
	}
	out, err := runShell(ctx, cmd, nil, env, 60)
	if out != "" {
		sess.Notify(core.NotifyInfo, strings.TrimRight(out, "\n"))
	}
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("plugin command %s failed: %v", c.name, err))
		return err
	}
	return nil
}

type pluginTool struct {
	name        string
	description string
	schema      json.RawMessage
	run         string
	cwd         string
}

func newPluginTool(mt ManifestTool, cwd string) core.Tool {
	return &pluginTool{
		name:        mt.Name,
		description: mt.Description,
		schema:      mt.InputSchema,
		run:         mt.Run,
		cwd:         cwd,
	}
}

func (t *pluginTool) Name() string             { return t.name }
func (t *pluginTool) Description() string      { return t.description }
func (t *pluginTool) Schema() json.RawMessage  { return t.schema }

func (t *pluginTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	env := map[string]string{
		"CLAUDE_PLUGIN_DIR":  t.cwd,
		"CLAUDE_PLUGIN_TOOL": t.name,
	}
	out, err := runShell(ctx, t.run, []byte(input), env, 120)
	if err != nil {
		if out != "" {
			return out, fmt.Errorf("%s: %w", strings.TrimRight(out, "\n"), err)
		}
		return "", err
	}
	return out, nil
}

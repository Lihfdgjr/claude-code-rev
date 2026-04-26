package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/plugins"
)

type pluginsCmd struct {
	loader *plugins.Loader
}

func NewPlugins(loader *plugins.Loader) core.Command {
	return &pluginsCmd{loader: loader}
}

func (pluginsCmd) Name() string     { return "plugins" }
func (pluginsCmd) Synopsis() string { return "List and manage installed plugins" }

func (c pluginsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	fields := strings.Fields(strings.TrimSpace(args))
	if len(fields) == 0 {
		return c.list(sess)
	}
	switch fields[0] {
	case "enable":
		if len(fields) < 2 {
			sess.Notify(core.NotifyError, "usage: /plugins enable <name>")
			return nil
		}
		return pluginsEnable(sess, fields[1])
	case "disable":
		if len(fields) < 2 {
			sess.Notify(core.NotifyError, "usage: /plugins disable <name>")
			return nil
		}
		return pluginsDisable(sess, fields[1])
	case "show":
		if len(fields) < 2 {
			sess.Notify(core.NotifyError, "usage: /plugins show <name>")
			return nil
		}
		return pluginsShow(sess, fields[1])
	case "reload":
		return c.reload(sess)
	case "install", "remove":
		sess.Notify(core.NotifyInfo,
			"Manual install: place a plugin.json directory under ~/.claude/plugins/<name>/. Remove by deleting the directory.")
		return nil
	default:
		sess.Notify(core.NotifyError, fmt.Sprintf("unknown subcommand: %s", fields[0]))
		return nil
	}
}

func (c pluginsCmd) list(sess core.Session) error {
	if c.loader == nil {
		sess.Notify(core.NotifyInfo, "No plugin loader configured.")
		return nil
	}

	list, err := c.loader.Load()
	if err != nil {
		return err
	}
	if len(list) == 0 {
		sess.Notify(core.NotifyInfo,
			"No plugins discovered. Add a plugin at ~/.claude/plugins/<name>/plugin.json or .claude/plugins/<name>/plugin.json.")
		return nil
	}

	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })

	var b strings.Builder
	b.WriteString("Installed plugins:\n")
	for _, p := range list {
		version := p.Version
		if version == "" {
			version = "?"
		}
		fmt.Fprintf(&b, "  %s (v%s) commands=%d tools=%d\n", p.Name, version, len(p.Commands), len(p.Tools))
		if p.Description != "" {
			fmt.Fprintf(&b, "    %s\n", p.Description)
		}
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}

// reload re-runs plugin discovery. This is a partial reload: discovery
// is refreshed, but new tools/commands are NOT registered into the running
// CLI's tool/command registries — that requires a CLI restart.
func (c pluginsCmd) reload(sess core.Session) error {
	if c.loader == nil {
		sess.Notify(core.NotifyInfo, "No plugin loader configured.")
		return nil
	}
	list, err := c.loader.Load()
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("plugins reload: %v", err))
		return err
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf(
		"Plugins re-discovered: %d plugins. Restart the CLI to register their tools/commands.",
		len(list),
	))
	return nil
}

func pluginDir(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "plugins", name), nil
}

func pluginsEnable(sess core.Session, name string) error {
	dir, err := pluginDir(name)
	if err != nil {
		return err
	}
	disabled := dir + ".disabled"
	if _, err := os.Stat(disabled); err == nil {
		if err := os.Rename(disabled, dir); err != nil {
			return err
		}
		sess.Notify(core.NotifyInfo, fmt.Sprintf("Re-enabled plugin %q at %s", name, dir))
		return nil
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Plugin %q enabled at %s", name, dir))
	return nil
}

func pluginsDisable(sess core.Session, name string) error {
	dir, err := pluginDir(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			sess.Notify(core.NotifyError, fmt.Sprintf("plugin %q not found at %s", name, dir))
			return nil
		}
		return err
	}
	disabled := dir + ".disabled"
	if err := os.Rename(dir, disabled); err != nil {
		return err
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Disabled plugin %q (renamed to %s)", name, disabled))
	return nil
}

func pluginsShow(sess core.Session, name string) error {
	dir, err := pluginDir(name)
	if err != nil {
		return err
	}
	for _, candidate := range []string{dir, dir + ".disabled"} {
		manifest := filepath.Join(candidate, "plugin.json")
		data, err := os.ReadFile(manifest)
		if err != nil {
			continue
		}
		sess.Notify(core.NotifyInfo, fmt.Sprintf("Plugin: %s\nPath: %s\n\n%s", name, manifest, string(data)))
		return nil
	}
	sess.Notify(core.NotifyError, fmt.Sprintf("plugin.json not found for %q", name))
	return nil
}

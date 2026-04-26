package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claudecode/internal/core"
)

type initCmd struct{}

func NewInit() core.Command { return &initCmd{} }

func (initCmd) Name() string     { return "init" }
func (initCmd) Synopsis() string { return "Generate a starter CLAUDE.md for this project" }

func (initCmd) Run(ctx context.Context, args string, sess core.Session) error {
	cwd, err := os.Getwd()
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("Cannot determine working directory: %v", err))
		return err
	}

	target := filepath.Join(cwd, "CLAUDE.md")
	if _, err := os.Stat(target); err == nil {
		sess.Notify(core.NotifyWarn, fmt.Sprintf("CLAUDE.md already exists at %s; leaving it untouched.", target))
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		sess.Notify(core.NotifyError, fmt.Sprintf("Cannot stat CLAUDE.md: %v", err))
		return err
	}

	entries, err := os.ReadDir(cwd)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("Cannot read directory: %v", err))
		return err
	}

	var dirs []string
	manifests := map[string]bool{}
	manifestFiles := []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml"}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if !strings.HasPrefix(name, ".") {
				dirs = append(dirs, name)
			}
			continue
		}
		for _, m := range manifestFiles {
			if name == m {
				manifests[m] = true
			}
		}
	}

	projectName := filepath.Base(cwd)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", projectName)
	b.WriteString("This file gives Claude Code orientation when working in this repository.\n\n")

	b.WriteString("## Project Type\n\n")
	if len(manifests) == 0 {
		b.WriteString("No standard manifest detected.\n")
	} else {
		for _, m := range manifestFiles {
			if manifests[m] {
				fmt.Fprintf(&b, "- %s present\n", m)
			}
		}
	}
	b.WriteString("\n")

	b.WriteString("## Top-Level Directories\n\n")
	if len(dirs) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, d := range dirs {
			fmt.Fprintf(&b, "- `%s/`\n", d)
		}
	}
	b.WriteString("\n")

	b.WriteString("## Notes\n\n")
	b.WriteString("- Edit this file to capture build, test, and style conventions.\n")

	if err := os.WriteFile(target, []byte(b.String()), 0o644); err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("Failed to write CLAUDE.md: %v", err))
		return err
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Wrote starter CLAUDE.md at %s", target))
	return nil
}

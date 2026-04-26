package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claudecode/internal/core"
)

type memoryCmd struct{}

func NewMemory() core.Command { return &memoryCmd{} }

func (memoryCmd) Name() string     { return "memory" }
func (memoryCmd) Synopsis() string { return "List discovered CLAUDE.md memory files" }

func (memoryCmd) Run(ctx context.Context, args string, sess core.Session) error {
	var found []string

	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for {
			candidate := filepath.Join(dir, "CLAUDE.md")
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				found = append(found, candidate)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".claude", "CLAUDE.md")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			found = appendUnique(found, candidate)
		}
	}

	var b strings.Builder
	if len(found) == 0 {
		b.WriteString("No CLAUDE.md memory files discovered.")
	} else {
		b.WriteString("Discovered CLAUDE.md memory files:\n")
		for _, p := range found {
			fmt.Fprintf(&b, "  - %s\n", p)
		}
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}

func appendUnique(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

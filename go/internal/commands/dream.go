package commands

import (
	"context"
	"fmt"

	"claudecode/internal/core"
	"claudecode/internal/memory"
)

type dreamCmd struct {
	store     *memory.Store
	transport core.Transport
}

// NewDream returns the /dream command. transport may be nil to disable;
// store may be nil to fall back to a default store under ~/.claude/memory/.
func NewDream(store *memory.Store, transport core.Transport) core.Command {
	return &dreamCmd{store: store, transport: transport}
}

func (c *dreamCmd) Name() string     { return "dream" }
func (c *dreamCmd) Synopsis() string { return "extract durable memories from this conversation" }

func (c *dreamCmd) Run(ctx context.Context, args string, sess core.Session) error {
	if c.store == nil {
		sess.Notify(core.NotifyError, "/dream: memory store not configured")
		return nil
	}
	if c.transport == nil {
		sess.Notify(core.NotifyError, "/dream: API transport not configured")
		return nil
	}
	sess.Notify(core.NotifyInfo, "/dream: scanning conversation for memorable facts...")
	saved, err := memory.AutoDream(ctx, c.store, c.transport, sess.Model(), sess.History())
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("/dream: %v", err))
		return nil
	}
	if saved == 0 {
		sess.Notify(core.NotifyInfo, "/dream: nothing new worth remembering")
		return nil
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("/dream: saved %d new memor%s under %s", saved, plural(saved, "y", "ies"), c.store.Root()))
	return nil
}

func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return singular
	}
	return pluralForm
}

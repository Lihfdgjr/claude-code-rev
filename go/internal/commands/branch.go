package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"claudecode/internal/chat"
	"claudecode/internal/core"
)

type branchCmd struct{}

// NewBranch returns a /branch command that lists user-message indices or
// truncates the session at the given index.
func NewBranch() core.Command { return &branchCmd{} }

func (branchCmd) Name() string     { return "branch" }
func (branchCmd) Synopsis() string { return "List user-message indices, or branch off at <index>" }

func (branchCmd) Run(ctx context.Context, args string, sess core.Session) error {
	args = strings.TrimSpace(args)
	hist := sess.History()

	if args == "" {
		if len(hist) == 0 {
			sess.Notify(core.NotifyInfo, "branch: no messages")
			return nil
		}
		var b strings.Builder
		b.WriteString("User-message indices (use `/branch <index>` to fork before that message):\n")
		any := false
		for i, m := range hist {
			if m.Role != core.RoleUser {
				continue
			}
			any = true
			fmt.Fprintf(&b, "  %d | %s\n", i, summarizeUserMessage(m))
		}
		if !any {
			sess.Notify(core.NotifyInfo, "branch: no user messages")
			return nil
		}
		sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
		return nil
	}

	idx, err := strconv.Atoi(args)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("branch: invalid index %q", args))
		return err
	}
	prefix, err := chat.Branch(sess, idx)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("branch: %v", err))
		return err
	}
	sess.ResetHistory()
	for _, m := range prefix {
		sess.Append(m)
	}
	sess.Notify(core.NotifyInfo, fmt.Sprintf("branched at %d (%d messages retained)", idx, len(prefix)))
	return nil
}

func summarizeUserMessage(m core.Message) string {
	const maxLen = 80
	for _, b := range m.Blocks {
		t, ok := b.(core.TextBlock)
		if !ok {
			continue
		}
		s := strings.TrimSpace(t.Text)
		if s == "" {
			continue
		}
		if len(s) > maxLen {
			s = s[:maxLen] + "..."
		}
		return s
	}
	return "(no text)"
}

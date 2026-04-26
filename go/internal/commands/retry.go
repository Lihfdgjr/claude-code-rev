package commands

import (
	"context"
	"strings"

	"claudecode/internal/core"
)

type retryCmd struct{}

// NewRetry returns a /retry command that re-runs the last user prompt.
// It truncates any assistant turns that came after that prompt, then
// calls Session.Resubmit to push the same text back through the driver.
func NewRetry() core.Command { return &retryCmd{} }

func (retryCmd) Name() string     { return "retry" }
func (retryCmd) Synopsis() string { return "Re-run the last user message" }

func (retryCmd) Run(ctx context.Context, args string, sess core.Session) error {
	hist := sess.History()
	lastUser := -1
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role == core.RoleUser && containsUserText(hist[i]) {
			lastUser = i
			break
		}
	}
	if lastUser < 0 {
		sess.Notify(core.NotifyInfo, "retry: no prior user message")
		return nil
	}

	text := extractUserText(hist[lastUser])
	if strings.TrimSpace(text) == "" {
		sess.Notify(core.NotifyInfo, "retry: last user message has no text")
		return nil
	}

	// Drop the trailing user message and everything after it; Resubmit
	// will re-append it via the normal Driver.Submit path.
	prefix := make([]core.Message, lastUser)
	copy(prefix, hist[:lastUser])
	sess.ResetHistory()
	for _, m := range prefix {
		sess.Append(m)
	}

	sess.Notify(core.NotifyInfo, "retry: resubmitting last prompt")
	sess.Resubmit(text)
	return nil
}

func containsUserText(m core.Message) bool {
	for _, b := range m.Blocks {
		if t, ok := b.(core.TextBlock); ok {
			if strings.TrimSpace(t.Text) != "" {
				return true
			}
		}
	}
	return false
}

func extractUserText(m core.Message) string {
	var parts []string
	for _, b := range m.Blocks {
		if t, ok := b.(core.TextBlock); ok {
			s := strings.TrimSpace(t.Text)
			if s != "" {
				parts = append(parts, s)
			}
		}
	}
	return strings.Join(parts, "\n")
}

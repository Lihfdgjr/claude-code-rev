package commands

import (
	"context"
	"fmt"

	"claudecode/internal/chat"
	"claudecode/internal/core"
)

type tokensCmd struct{}

func NewTokens() core.Command { return &tokensCmd{} }

func (tokensCmd) Name() string     { return "tokens" }
func (tokensCmd) Synopsis() string { return "Show token usage and context budget" }

func (tokensCmd) Run(ctx context.Context, args string, sess core.Session) error {
	model := sess.Model()
	hist := sess.History()
	cum := sess.CumulativeUsage()

	live := chat.Usage(hist)
	limit := chat.ContextLimit(model)
	pct := 0.0
	if limit > 0 {
		pct = float64(live) / float64(limit) * 100
	}

	msg := fmt.Sprintf(
		"Model: %s\nCumulative input:  %d\nCumulative output: %d\nCache read:        %d\nCache write:       %d\nHistory estimate:  %d tokens\nContext limit:     %d tokens\nUsed:              %.1f%%",
		model,
		cum.InputTokens,
		cum.OutputTokens,
		cum.CacheReadTokens,
		cum.CacheCreationTokens,
		live,
		limit,
		pct,
	)
	sess.Notify(core.NotifyInfo, msg)
	return nil
}

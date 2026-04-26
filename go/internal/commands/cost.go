package commands

import (
	"context"
	"fmt"

	"claudecode/internal/core"
	"claudecode/internal/pricing"
)

type costCmd struct{}

func NewCost() core.Command { return &costCmd{} }

func (costCmd) Name() string     { return "cost" }
func (costCmd) Synopsis() string { return "Show estimated session cost" }

func (costCmd) Run(ctx context.Context, args string, sess core.Session) error {
	u := sess.CumulativeUsage()
	model := sess.Model()
	total := pricing.Estimate(model, u)

	msg := fmt.Sprintf(
		"Model: %s\nInput tokens:  %d\nOutput tokens: %d\nCache read:    %d\nCache write:   %d\nTotal cost:    %s",
		model,
		u.InputTokens,
		u.OutputTokens,
		u.CacheReadTokens,
		u.CacheCreationTokens,
		pricing.FormatUSD(total),
	)
	sess.Notify(core.NotifyInfo, msg)
	return nil
}

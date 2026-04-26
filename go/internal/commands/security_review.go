package commands

import (
	"context"

	"claudecode/internal/core"
)

type securityReviewCmd struct{}

func NewSecurityReview() core.Command { return &securityReviewCmd{} }

func (securityReviewCmd) Name() string     { return "security-review" }
func (securityReviewCmd) Synopsis() string { return "Run a security review of pending changes" }

func (securityReviewCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo,
		"Security review pending - checks for hardcoded secrets, unsafe patterns, etc. Run manually for now.")
	return nil
}

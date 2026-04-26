package commands

import (
	"context"

	"claudecode/internal/core"
)

type privacySettingsCmd struct{}

func NewPrivacySettings() core.Command { return &privacySettingsCmd{} }

func (privacySettingsCmd) Name() string     { return "privacy-settings" }
func (privacySettingsCmd) Synopsis() string { return "Show telemetry and privacy status" }

func (privacySettingsCmd) Run(ctx context.Context, args string, sess core.Session) error {
	sess.Notify(core.NotifyInfo,
		"Telemetry is currently disabled in this build. Edit ~/.claude/settings.json to configure.")
	return nil
}

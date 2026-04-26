package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"claudecode/internal/core"
)

type importCmd struct{}

func NewImport() core.Command { return &importCmd{} }

func (importCmd) Name() string     { return "import" }
func (importCmd) Synopsis() string { return "Import a file's contents as a user message" }

func (importCmd) Run(ctx context.Context, args string, sess core.Session) error {
	path := strings.TrimSpace(args)
	if path == "" {
		sess.Notify(core.NotifyWarn, "Usage: /import <path>")
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("read %s: %v", path, err))
		return nil
	}
	sess.Append(core.Message{
		Role:   core.RoleUser,
		Blocks: []core.Block{core.TextBlock{Text: string(data)}},
	})
	sess.Notify(core.NotifyInfo, fmt.Sprintf("Imported %d bytes from %s", len(data), path))
	return nil
}

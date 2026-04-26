package commands

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"claudecode/internal/core"
)

const (
	shellCmdTimeout = 30 * time.Second
	shellCmdMaxOut  = 5000
)

type shellCmd struct{}

func NewShell() core.Command { return &shellCmd{} }

func (shellCmd) Name() string     { return "shell" }
func (shellCmd) Synopsis() string { return "Run a shell command (alias /sh)" }

func (shellCmd) Run(ctx context.Context, args string, sess core.Session) error {
	cmd := strings.TrimSpace(args)
	if cmd == "" {
		sess.Notify(core.NotifyWarn, "Usage: /shell <command>")
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, shellCmdTimeout)
	defer cancel()

	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.CommandContext(cctx, "cmd", "/C", cmd)
	} else {
		c = exec.CommandContext(cctx, "sh", "-c", cmd)
	}
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf
	runErr := c.Run()
	out := buf.String()
	if len(out) > shellCmdMaxOut {
		out = out[:shellCmdMaxOut]
	}
	if runErr != nil {
		if cctx.Err() == context.DeadlineExceeded {
			sess.Notify(core.NotifyError, fmt.Sprintf("/shell timeout after %s\n%s", shellCmdTimeout, out))
		} else {
			sess.Notify(core.NotifyError, fmt.Sprintf("/shell error: %v\n%s", runErr, out))
		}
		return nil
	}
	if out == "" {
		sess.Notify(core.NotifyInfo, "(no output)")
	} else {
		sess.Notify(core.NotifyInfo, out)
	}
	return nil
}

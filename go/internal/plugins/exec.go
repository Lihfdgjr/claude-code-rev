package plugins

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"runtime"
	"time"
)

func runShell(ctx context.Context, cmd string, stdin []byte, env map[string]string, timeoutSec int) (string, error) {
	if cmd == "" {
		return "", errors.New("empty command")
	}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.CommandContext(cctx, "cmd", "/C", cmd)
	} else {
		c = exec.CommandContext(cctx, "sh", "-c", cmd)
	}

	if len(env) > 0 {
		base := append([]string(nil), c.Env...)
		for k, v := range env {
			base = append(base, k+"="+v)
		}
		c.Env = base
	}

	if len(stdin) > 0 {
		c.Stdin = bytes.NewReader(stdin)
	}

	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out

	err := c.Run()
	return out.String(), err
}

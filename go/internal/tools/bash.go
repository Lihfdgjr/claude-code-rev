package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"claudecode/internal/core"
)

const (
	bashDefaultTimeoutMs = 120000
	bashMaxTimeoutMs     = 600000
	bashOutputCap        = 30000
)

type bashTool struct{}

type bashInput struct {
	Command     string `json:"command"`
	TimeoutMs   int    `json:"timeout_ms,omitempty"`
	Description string `json:"description,omitempty"`
}

func NewBash() core.Tool { return &bashTool{} }

func (bashTool) Name() string { return "Bash" }

func (bashTool) Description() string {
	return "Execute a shell command with a timeout. Returns combined stdout/stderr and exit code."
}

func (bashTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "command": {"type": "string", "description": "Shell command to execute"},
    "timeout_ms": {"type": "integer", "description": "Timeout in milliseconds (max 600000)", "minimum": 1, "maximum": 600000},
    "description": {"type": "string", "description": "Optional human-readable description"}
  },
  "required": ["command"],
  "additionalProperties": false
}`)
}

func (bashTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in bashInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	timeoutMs := in.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = bashDefaultTimeoutMs
	}
	if timeoutMs > bashMaxTimeoutMs {
		timeoutMs = bashMaxTimeoutMs
	}

	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cctx, "cmd", "/C", in.Command)
	} else {
		cmd = exec.CommandContext(cctx, "sh", "-c", in.Command)
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}
	if cctx.Err() == context.DeadlineExceeded {
		exitCode = -1
		buf.WriteString(fmt.Sprintf("\n[command timed out after %dms]", timeoutMs))
	}

	out := buf.String()
	if len(out) > bashOutputCap {
		out = out[:bashOutputCap] + fmt.Sprintf("\n... [truncated, %d bytes total]", len(out))
	}
	return fmt.Sprintf("exit_code: %d\n%s", exitCode, out), nil
}

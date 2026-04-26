package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const maxStdoutBytes = 64 * 1024

type Runner struct {
	cfg Config
}

func New(cfg Config) *Runner {
	if cfg == nil {
		cfg = Config{}
	}
	return &Runner{cfg: cfg}
}

type hookOutput struct {
	Block            bool            `json:"block"`
	Reason           string          `json:"reason"`
	ReplacementInput json.RawMessage `json:"replacement_input"`
}

func (r *Runner) Run(ctx context.Context, ev Event) (Decision, error) {
	specs, ok := r.cfg[ev.Name]
	if !ok || len(specs) == 0 {
		return Decision{}, nil
	}

	isToolEvent := ev.Name == PreToolUse || ev.Name == PostToolUse

	var decision Decision
	for _, spec := range specs {
		if isToolEvent && spec.Matcher != "" {
			re, err := regexp.Compile(spec.Matcher)
			if err != nil {
				continue
			}
			if !re.MatchString(ev.ToolName) {
				continue
			}
		}
		if spec.Type != "" && spec.Type != "command" {
			continue
		}
		if spec.Command == "" {
			continue
		}

		out, exitCode, stderr, runErr := runOne(ctx, spec, ev)
		if runErr != nil {
			return decision, runErr
		}

		parsed := hookOutput{}
		jsonErr := json.Unmarshal(bytes.TrimSpace(out), &parsed)
		if jsonErr == nil {
			if parsed.Block {
				decision.Block = true
				decision.Reason = parsed.Reason
				if len(parsed.ReplacementInput) > 0 {
					decision.ReplacementInput = parsed.ReplacementInput
				}
				return decision, nil
			}
			if len(parsed.ReplacementInput) > 0 {
				decision.ReplacementInput = parsed.ReplacementInput
			}
			continue
		}

		if exitCode != 0 {
			reason := strings.TrimSpace(stderr)
			if reason == "" {
				reason = fmt.Sprintf("hook exited with status %d", exitCode)
			}
			decision.Block = true
			decision.Reason = reason
			return decision, nil
		}
	}

	return decision, nil
}

func runOne(ctx context.Context, spec HookSpec, ev Event) ([]byte, int, string, error) {
	timeout := time.Duration(spec.Timeout) * time.Second
	if spec.Timeout <= 0 {
		timeout = 30 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cctx, "cmd", "/C", spec.Command)
	} else {
		cmd = exec.CommandContext(cctx, "sh", "-c", spec.Command)
	}

	payload, err := json.Marshal(struct {
		Name       EventName       `json:"event"`
		ToolName   string          `json:"tool_name,omitempty"`
		ToolInput  json.RawMessage `json:"tool_input,omitempty"`
		ToolOutput string          `json:"tool_output,omitempty"`
		UserText   string          `json:"user_text,omitempty"`
		SessionID  string          `json:"session_id,omitempty"`
	}{
		Name:       ev.Name,
		ToolName:   ev.ToolName,
		ToolInput:  ev.ToolInput,
		ToolOutput: ev.ToolOutput,
		UserText:   ev.UserText,
		SessionID:  ev.SessionID,
	})
	if err != nil {
		return nil, 0, "", err
	}
	cmd.Stdin = bytes.NewReader(payload)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &stdoutBuf, remaining: maxStdoutBytes}
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
			runErr = nil
		} else if cctx.Err() == context.DeadlineExceeded {
			return stdoutBuf.Bytes(), -1, "hook timed out", nil
		}
	}

	return stdoutBuf.Bytes(), exitCode, stderrBuf.String(), runErr
}

type limitedWriter struct {
	w         io.Writer
	remaining int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.remaining <= 0 {
		return len(p), nil
	}
	if len(p) > l.remaining {
		_, err := l.w.Write(p[:l.remaining])
		l.remaining = 0
		if err != nil {
			return 0, err
		}
		return len(p), nil
	}
	n, err := l.w.Write(p)
	l.remaining -= n
	return n, err
}

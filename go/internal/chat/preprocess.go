package chat

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"claudecode/internal/core"
)

const (
	maxShellOutput  = 30000
	maxAttachBytes  = 200 * 1024
	shellTimeout    = 30 * time.Second
)

var mentionRe = regexp.MustCompile(`@([^\s]+)`)

// ExpandUserInput preprocesses a user message: lines beginning with `!`
// are executed as shell commands (output substituted inline) and `@path`
// tokens are auto-attached as DocumentBlocks.
func ExpandUserInput(text string, cwd string) (string, []core.Block, error) {
	var attachments []core.Block

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "!") {
			cmd := strings.TrimSpace(line[1:])
			if cmd == "" {
				continue
			}
			out := runShellCapture(cmd)
			if len(out) > maxShellOutput {
				out = out[:maxShellOutput]
			}
			lines[i] = fmt.Sprintf("Output of `!%s`:\n```\n%s\n```", cmd, out)
		}
	}
	text = strings.Join(lines, "\n")

	text = mentionRe.ReplaceAllStringFunc(text, func(tok string) string {
		path := tok[1:]
		full := path
		if !filepath.IsAbs(full) {
			full = filepath.Join(cwd, path)
		}
		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			return tok
		}
		size := info.Size()
		if size > maxAttachBytes {
			return tok
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return tok
		}
		base := filepath.Base(full)
		attachments = append(attachments, core.DocumentBlock{
			Source:    base64.StdEncoding.EncodeToString(data),
			MediaType: "text/plain",
			Title:     base,
		})
		return fmt.Sprintf("[file: %s]", base)
	})

	return text, attachments, nil
}

func runShellCapture(cmd string) string {
	ctx, cancel := context.WithTimeout(context.Background(), shellTimeout)
	defer cancel()

	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.CommandContext(ctx, "cmd", "/C", cmd)
	} else {
		c = exec.CommandContext(ctx, "sh", "-c", cmd)
	}
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf
	if err := c.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Fprintf(&buf, "\n[timeout after %s]", shellTimeout)
		} else {
			fmt.Fprintf(&buf, "\n[exit error: %v]", err)
		}
	}
	return buf.String()
}

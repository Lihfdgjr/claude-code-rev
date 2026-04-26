package commands

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"claudecode/internal/core"
)

const maxClipboardChars = 50000

type clipboardCmd struct{}

func NewClipboard() core.Command { return &clipboardCmd{} }

func (clipboardCmd) Name() string     { return "clipboard" }
func (clipboardCmd) Synopsis() string { return "Paste clipboard contents into the next message" }

func (clipboardCmd) Run(ctx context.Context, args string, sess core.Session) error {
	text, err := readClipboard(ctx)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("clipboard: %v", err))
		return nil
	}
	if text == "" {
		sess.Notify(core.NotifyWarn, "clipboard is empty")
		return nil
	}
	if len(text) > maxClipboardChars {
		text = text[:maxClipboardChars]
		sess.Notify(core.NotifyWarn, fmt.Sprintf("clipboard truncated to %d chars", maxClipboardChars))
	}
	sess.Attach(core.TextBlock{Text: "Pasted clipboard:\n" + text})
	sess.Notify(core.NotifyInfo, fmt.Sprintf("attached clipboard text (%d chars); will be sent with your next message.", len(text)))
	return nil
}

func readClipboard(ctx context.Context) (string, error) {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Get-Clipboard").Output()
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(out), "\r\n"), nil
	case "darwin":
		out, err := exec.CommandContext(ctx, "pbpaste").Output()
		if err != nil {
			return "", err
		}
		return string(out), nil
	case "linux":
		if out, err := exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-o").Output(); err == nil {
			return string(out), nil
		}
		if out, err := exec.CommandContext(ctx, "wl-paste").Output(); err == nil {
			return string(out), nil
		}
		return "", errors.New("xclip or wl-paste not available")
	}
	return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
}

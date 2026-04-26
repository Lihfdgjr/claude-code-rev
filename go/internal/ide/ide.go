package ide

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ErrNotSupported is returned by operations that have no real implementation
// in this build (e.g. SelectedText without an IDE protocol).
var ErrNotSupported = errors.New("ide: not supported in this build")

// IDE is the editor-integration surface used by commands and tools.
type IDE interface {
	Name() string
	OpenFile(ctx context.Context, path string, line int) error
	Diagnostics(ctx context.Context) ([]string, error)
	SelectedText(ctx context.Context) (string, error)
}

type detected struct {
	name string
}

// Detect inspects well-known environment variables and returns an IDE
// implementation for the detected editor (or a generic fallback).
func Detect() IDE {
	if v := strings.TrimSpace(os.Getenv("TERM_PROGRAM")); v != "" {
		switch strings.ToLower(v) {
		case "vscode":
			return detected{name: "vscode"}
		case "apple_terminal":
			return detected{name: "apple-terminal"}
		case "iterm.app":
			return detected{name: "iterm2"}
		default:
			return detected{name: v}
		}
	}
	if os.Getenv("VSCODE_INJECTION") != "" || os.Getenv("VSCODE_PID") != "" {
		return detected{name: "vscode"}
	}
	if v := os.Getenv("JETBRAINS_IDE"); v != "" {
		return detected{name: v}
	}
	return detected{name: "generic"}
}

func (d detected) Name() string { return d.name }

func (d detected) OpenFile(ctx context.Context, path string, line int) error {
	switch strings.ToLower(d.name) {
	case "vscode":
		target := path
		if line > 0 {
			target = fmt.Sprintf("%s:%d", path, line)
		}
		return exec.CommandContext(ctx, "code", "--goto", target).Run()
	case "generic":
		return errors.New("ide: no IDE detected")
	default:
		// Best-effort JetBrains launch; the binary name varies per product
		// (idea, pycharm, webstorm, ...). We try `idea` first and ignore
		// failure since callers treat OpenFile as advisory.
		args := []string{}
		if line > 0 {
			args = append(args, "--line", strconv.Itoa(line))
		}
		args = append(args, path)
		_ = exec.CommandContext(ctx, "idea", args...).Run()
		return nil
	}
}

// Diagnostics is a no-op for now: we cannot query VSCode's problem list from
// the CLI without the IDE extension protocol. Returns an empty slice so
// callers can treat the result uniformly.
func (d detected) Diagnostics(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (d detected) SelectedText(ctx context.Context) (string, error) {
	return "", ErrNotSupported
}

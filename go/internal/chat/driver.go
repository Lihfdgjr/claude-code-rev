package chat

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"claudecode/internal/core"
	"claudecode/internal/hooks"
	"claudecode/internal/sessions"
	"claudecode/internal/telemetry"
)

type Config struct {
	Transport    core.Transport
	Tools        core.ToolRegistry
	Commands     core.CommandRegistry
	Permissions  core.PermissionGate
	Hooks        *hooks.Runner
	Transcript   *sessions.Recorder
	Model        string
	SystemPrompt string
	SessionID    string
	OnTurnDone   func(history []core.Message)
	OnPostTurn   func(history []core.Message)
	AutoCompact  *AutoCompact
}

type driver struct {
	cfg       Config
	sess      *session
	notifChan chan core.UIEvent

	mu        sync.Mutex
	cancel    context.CancelFunc
	titleOnce sync.Once
}

func NewDriver(cfg Config) core.Driver {
	d := &driver{
		cfg:       cfg,
		notifChan: make(chan core.UIEvent, 64),
	}
	d.sess = &session{
		model:        cfg.Model,
		systemPrompt: cfg.SystemPrompt,
		transport:    cfg.Transport,
	}
	d.sess.notifier = func(level core.NotifyLevel, msg string) {
		ev := core.UIStatusEvent{Level: level, Text: msg}
		select {
		case d.notifChan <- ev:
		default:
		}
	}
	d.sess.resubmit = func(text string) { d.Submit(text) }
	d.sess.cancel = d.Cancel
	return d
}

func (d *driver) Session() core.Session              { return d.sess }
func (d *driver) Snapshot() []core.Message           { return d.sess.History() }
func (d *driver) Notifications() <-chan core.UIEvent { return d.notifChan }
func (d *driver) Commands() core.CommandRegistry     { return d.cfg.Commands }
func (d *driver) Tools() core.ToolRegistry           { return d.cfg.Tools }

func (d *driver) Cancel() {
	d.mu.Lock()
	if d.cancel != nil {
		d.cancel()
	}
	d.mu.Unlock()
}

func (d *driver) Submit(text string) <-chan core.UIEvent {
	out := make(chan core.UIEvent, 32)

	text = strings.TrimSpace(text)
	if text == "" {
		close(out)
		return out
	}

	// UserPromptSubmit hook may block or rewrite the user text.
	if d.cfg.Hooks != nil {
		dec, _ := d.cfg.Hooks.Run(context.Background(), hooks.Event{
			Name:      hooks.UserPromptSubmit,
			UserText:  text,
			SessionID: d.cfg.SessionID,
		})
		if dec.Block {
			go func() {
				defer close(out)
				out <- core.UIErrorEvent{Err: fmt.Errorf("blocked by hook: %s", dec.Reason)}
			}()
			return out
		}
		if len(dec.ReplacementInput) > 0 {
			text = strings.TrimSpace(string(dec.ReplacementInput))
		}
	}

	cwd, _ := os.Getwd()
	expanded, autoAttach, err := ExpandUserInput(text, cwd)
	if err != nil {
		go func() {
			defer close(out)
			out <- core.UIErrorEvent{Err: fmt.Errorf("preprocess: %w", err)}
		}()
		return out
	}
	text = expanded
	for _, b := range autoAttach {
		d.sess.Attach(b)
	}

	blocks := d.sess.DrainAttachments()
	attachmentCount := len(blocks)
	blocks = append(blocks, core.TextBlock{Text: text})
	d.sess.Append(core.Message{
		Role:   core.RoleUser,
		Blocks: blocks,
	})

	if d.cfg.Transcript != nil {
		_ = d.cfg.Transcript.Write("user_message", map[string]interface{}{
			"text":              text,
			"attachments_count": attachmentCount,
		})
	}

	telemetry.LogGlobal("turn.start", map[string]interface{}{
		"text_len":         len(text),
		"session_id":       d.cfg.SessionID,
		"attachment_count": attachmentCount,
	})

	ctx, cancel := context.WithCancel(context.Background())
	d.mu.Lock()
	if d.cancel != nil {
		d.cancel()
	}
	d.cancel = cancel
	d.mu.Unlock()

	go func() {
		defer close(out)
		defer func() {
			d.mu.Lock()
			d.cancel = nil
			d.mu.Unlock()
		}()
		err := d.runLoop(ctx, out)
		if err != nil && ctx.Err() == nil {
			select {
			case out <- core.UIErrorEvent{Err: err}:
			default:
			}
		}
	}()

	return out
}

// handleTurnDone fires the user-supplied OnTurnDone callback and, on the
// first completed turn, kicks off background auto-titling.
func (d *driver) handleTurnDone(stopReason string) {
	hist := d.sess.History()
	if d.cfg.Transcript != nil {
		blockCount := 0
		for i := len(hist) - 1; i >= 0; i-- {
			if hist[i].Role == core.RoleAssistant {
				blockCount = len(hist[i].Blocks)
				break
			}
		}
		_ = d.cfg.Transcript.Write("assistant_message", map[string]interface{}{
			"block_count": blockCount,
			"stop_reason": stopReason,
		})
	}
	if d.cfg.OnTurnDone != nil {
		d.cfg.OnTurnDone(hist)
	}
	if d.cfg.OnPostTurn != nil {
		d.cfg.OnPostTurn(hist)
	}
	if d.cfg.AutoCompact != nil {
		d.cfg.AutoCompact.MaybeRun(context.Background(), d.sess)
	}
	if len(hist) < 2 || d.sess.Title() != "" {
		return
	}
	d.titleOnce.Do(func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			snapshot := d.sess.History()
			title, err := GenerateTitle(ctx, d.cfg.Transport, d.sess.Model(), snapshot)
			if err == nil && title != "" {
				d.sess.SetTitle(title)
			}
		}()
	})
}

func (d *driver) RunCommand(line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	name, rest, _ := strings.Cut(line, " ")
	cmd, ok := d.cfg.Commands.Get(name)
	if !ok {
		return fmt.Errorf("%w: /%s", core.ErrCommandNotFound, name)
	}
	args := strings.TrimSpace(rest)
	telemetry.LogGlobal("command.invoke", map[string]interface{}{
		"command":    name,
		"args_len":   len(args),
		"session_id": d.cfg.SessionID,
	})
	return cmd.Run(context.Background(), args, d.sess)
}

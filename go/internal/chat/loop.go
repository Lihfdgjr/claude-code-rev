package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"claudecode/internal/core"
	"claudecode/internal/hooks"
	"claudecode/internal/telemetry"
)

const maxLoopIterations = 50

type pendingBlock struct {
	kind     core.BlockKind
	text     strings.Builder
	toolID   string
	toolName string
}

func (d *driver) runLoop(ctx context.Context, out chan<- core.UIEvent) error {
	for i := 0; i < maxLoopIterations; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		opts := core.CallOptions{
			Model:        d.sess.Model(),
			SystemPrompt: d.sess.SystemPrompt(),
			Tools:        d.cfg.Tools.All(),
			MaxTokens:    8192,
			Thinking:     ThinkingEnabled.Load(),
		}

		hist := trimIfOverBudget(d.sess.History(), d.sess.Model())
		ch, err := StreamWithRetry(ctx, d.cfg.Transport, opts, hist, 4)
		if err != nil {
			return fmt.Errorf("stream: %w", err)
		}

		var assistantBlocks []core.Block
		pending := map[int]*pendingBlock{}
		var stopReason string
		var usage core.Usage

		for ev := range ch {
			if err := ctx.Err(); err != nil {
				return err
			}
			switch e := ev.(type) {
			case core.MessageStartEvent:
			case core.TextDeltaEvent:
				pb := pending[e.Index]
				if pb == nil {
					pb = &pendingBlock{kind: core.KindText}
					pending[e.Index] = pb
				}
				pb.text.WriteString(e.Text)
				if !forward(ctx, out, core.UIAssistantTextDeltaEvent{Text: e.Text}) {
					return ctx.Err()
				}
			case core.ThinkingDeltaEvent:
				pb := pending[e.Index]
				if pb == nil {
					pb = &pendingBlock{kind: core.KindThinking}
					pending[e.Index] = pb
				}
				pb.text.WriteString(e.Text)
				if !forward(ctx, out, core.UIThinkingDeltaEvent{Text: e.Text}) {
					return ctx.Err()
				}
			case core.ToolUseStartEvent:
				pending[e.Index] = &pendingBlock{
					kind:     core.KindToolUse,
					toolID:   e.ID,
					toolName: e.Name,
				}
			case core.ToolInputDeltaEvent:
				pb := pending[e.Index]
				if pb == nil {
					pb = &pendingBlock{kind: core.KindToolUse}
					pending[e.Index] = pb
				}
				pb.text.WriteString(e.JSONPart)
			case core.BlockEndEvent:
				pb := pending[e.Index]
				if pb == nil {
					continue
				}
				switch pb.kind {
				case core.KindText:
					assistantBlocks = append(assistantBlocks, core.TextBlock{Text: pb.text.String()})
				case core.KindThinking:
					assistantBlocks = append(assistantBlocks, core.ThinkingBlock{Text: pb.text.String()})
				case core.KindToolUse:
					raw := pb.text.String()
					if raw == "" {
						raw = "{}"
					}
					assistantBlocks = append(assistantBlocks, core.ToolUseBlock{
						ID:    pb.toolID,
						Name:  pb.toolName,
						Input: json.RawMessage(raw),
					})
				}
				delete(pending, e.Index)
			case core.MessageEndEvent:
				stopReason = e.StopReason
				usage = e.Usage
				d.sess.AddUsage(usage)
				telemetry.LogGlobal("turn.done", map[string]interface{}{
					"tokens_input":  usage.InputTokens,
					"tokens_output": usage.OutputTokens,
					"model":         d.sess.Model(),
					"stop_reason":   stopReason,
					"session_id":    d.cfg.SessionID,
				})
			case core.ErrorEvent:
				telemetry.LogGlobal("turn.error", map[string]interface{}{
					"error":      e.Err.Error(),
					"session_id": d.cfg.SessionID,
				})
				if d.cfg.Transcript != nil {
					_ = d.cfg.Transcript.Write("error", map[string]interface{}{
						"msg": e.Err.Error(),
					})
				}
				return e.Err
			}
		}

		if len(assistantBlocks) > 0 {
			d.sess.Append(core.Message{
				Role:   core.RoleAssistant,
				Blocks: assistantBlocks,
			})
		}

		var toolUses []core.ToolUseBlock
		for _, b := range assistantBlocks {
			if tu, ok := b.(core.ToolUseBlock); ok {
				toolUses = append(toolUses, tu)
			}
		}

		if len(toolUses) == 0 || stopReason != "tool_use" {
			d.runHook(ctx, hooks.Event{Name: hooks.Stop, SessionID: d.cfg.SessionID})
			d.handleTurnDone(stopReason)
			forward(ctx, out, core.UITurnDoneEvent{StopReason: stopReason, Usage: usage})
			return nil
		}

		var resultBlocks []core.Block
		for _, tu := range toolUses {
			preview := string(tu.Input)
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			forward(ctx, out, core.UIToolStartEvent{ID: tu.ID, Name: tu.Name, Input: preview})

			output, isErr := d.executeTool(ctx, tu, out)

			forward(ctx, out, core.UIToolResultEvent{ID: tu.ID, Output: output, IsError: isErr})

			resultBlocks = append(resultBlocks, core.ToolResultBlock{
				UseID:   tu.ID,
				Content: output,
				IsError: isErr,
			})
		}

		d.sess.Append(core.Message{
			Role:   core.RoleUser,
			Blocks: resultBlocks,
		})
	}

	return errors.New("max loop iterations exceeded")
}

func forward(ctx context.Context, out chan<- core.UIEvent, ev core.UIEvent) bool {
	select {
	case out <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

func (d *driver) executeTool(ctx context.Context, tu core.ToolUseBlock, out chan<- core.UIEvent) (string, bool) {
	tool, ok := d.cfg.Tools.Get(tu.Name)
	if !ok {
		return fmt.Sprintf("tool not found: %s", tu.Name), true
	}

	input := []byte(tu.Input)

	// Pre-tool hooks may block or rewrite input.
	if dec := d.runHook(ctx, hooks.Event{
		Name:      hooks.PreToolUse,
		ToolName:  tu.Name,
		ToolInput: input,
		SessionID: d.cfg.SessionID,
	}); dec.Block {
		return fmt.Sprintf("blocked by hook: %s", dec.Reason), true
	} else if len(dec.ReplacementInput) > 0 {
		input = dec.ReplacementInput
	}

	if d.cfg.Permissions != nil {
		decision, reason := d.cfg.Permissions.Check(ctx, core.PermissionRequest{
			Tool:  tu.Name,
			Input: input,
		})
		switch decision {
		case core.PermissionDeny:
			return fmt.Sprintf("permission denied: %s", reason), true
		case core.PermissionAsk:
			reply := make(chan core.PermissionResponse, 1)
			ev := core.UIPermissionPromptEvent{
				Tool:       tu.Name,
				InputJSON:  string(input),
				ReplyChan:  reply,
				RememberOK: true,
			}
			if !forward(ctx, out, ev) {
				return "cancelled", true
			}
			select {
			case answered := <-reply:
				if answered.Decision == core.PermissionDeny {
					return fmt.Sprintf("permission denied by user: %s", reason), true
				}
				if answered.Remember && answered.Decision == core.PermissionAllow {
					d.cfg.Permissions.AllowRuntime(tu.Name)
				}
			case <-ctx.Done():
				return "cancelled", true
			}
		}
	}

	if PlanModeActive.Load() && PlanWriteTools[tu.Name] {
		return fmt.Sprintf("blocked: plan mode active (tool %q would mutate state). Run /exit-plan-mode or invoke ExitPlanMode.", tu.Name), true
	}

	start := time.Now()
	output, err := tool.Run(core.WithUIEvents(ctx, out), input)
	durMS := time.Since(start).Milliseconds()
	if err != nil {
		output = fmt.Sprintf("error: %v", err)
	}
	telemetry.LogGlobal("tool.done", map[string]interface{}{
		"tool":        tu.Name,
		"duration_ms": durMS,
		"error_bool":  err != nil,
		"session_id":  d.cfg.SessionID,
	})
	if d.cfg.Transcript != nil {
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		_ = d.cfg.Transcript.Write("tool_use", map[string]interface{}{
			"name":        tu.Name,
			"error":       errStr,
			"duration_ms": durMS,
		})
	}

	d.runHook(ctx, hooks.Event{
		Name:       hooks.PostToolUse,
		ToolName:   tu.Name,
		ToolInput:  input,
		ToolOutput: output,
		SessionID:  d.cfg.SessionID,
	})

	return output, err != nil
}

func (d *driver) runHook(ctx context.Context, ev hooks.Event) hooks.Decision {
	if d.cfg.Hooks == nil {
		return hooks.Decision{}
	}
	dec, _ := d.cfg.Hooks.Run(ctx, ev)
	return dec
}

// trimIfOverBudget returns a defensive copy of history with the oldest
// tool_result blocks neutered when context use exceeds 95%, stopping once
// usage drops below 85%. It never removes text or tool_use blocks so the
// model's reasoning trail stays intact.
func trimIfOverBudget(history []core.Message, model string) []core.Message {
	if BudgetPercent(history, model) <= 0.95 {
		return history
	}

	out := make([]core.Message, len(history))
	for i, m := range history {
		blocks := make([]core.Block, len(m.Blocks))
		copy(blocks, m.Blocks)
		out[i] = core.Message{Role: m.Role, Blocks: blocks}
	}

	for i := range out {
		for j, b := range out[i].Blocks {
			tr, ok := b.(core.ToolResultBlock)
			if !ok {
				continue
			}
			if tr.Content == "[trimmed]" {
				continue
			}
			out[i].Blocks[j] = core.ToolResultBlock{
				UseID:   tr.UseID,
				Content: "[trimmed]",
				IsError: tr.IsError,
			}
			if BudgetPercent(out, model) <= 0.85 {
				return out
			}
		}
	}
	return out
}

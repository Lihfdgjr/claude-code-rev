package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"claudecode/internal/core"
)

type SpawnerConfig struct {
	Transport   core.Transport
	Tools       core.ToolRegistry
	Permissions core.PermissionGate
	Model       string
	MaxTurns    int
}

type spawner struct {
	cfg SpawnerConfig
}

func NewSpawner(cfg SpawnerConfig) core.Spawner {
	if cfg.MaxTurns <= 0 {
		cfg.MaxTurns = 25
	}
	return &spawner{cfg: cfg}
}

func (s *spawner) Spawn(ctx context.Context, opts core.SpawnOptions) (string, error) {
	depth := core.SubagentDepth(ctx)
	if depth >= core.MaxSubagentDepth {
		return "", errors.New("subagent recursion limit reached")
	}
	ctx = core.WithSubagentDepth(ctx, depth+1)

	history := []core.Message{
		{Role: core.RoleUser, Blocks: []core.Block{core.TextBlock{Text: opts.Prompt}}},
	}

	sysPrompt := opts.SystemPrompt
	if sysPrompt == "" {
		sysPrompt = "You are a focused sub-agent. Use the provided tools to accomplish the task and return a concise summary of what you did."
	}

	model := opts.Model
	if model == "" {
		model = s.cfg.Model
	}

	maxTurns := opts.MaxTurns
	if maxTurns <= 0 {
		maxTurns = s.cfg.MaxTurns
	}

	availTools := opts.Tools
	if len(availTools) == 0 {
		if len(opts.AllowedTools) > 0 && s.cfg.Tools != nil {
			allow := make(map[string]struct{}, len(opts.AllowedTools))
			for _, n := range opts.AllowedTools {
				allow[n] = struct{}{}
			}
			for _, t := range s.cfg.Tools.All() {
				if _, ok := allow[t.Name()]; ok {
					availTools = append(availTools, t)
				}
			}
		} else {
			availTools = s.cfg.Tools.All()
		}
	}

	var finalText strings.Builder

	for i := 0; i < maxTurns; i++ {
		if err := ctx.Err(); err != nil {
			return finalText.String(), err
		}

		ch, err := s.cfg.Transport.Stream(ctx, core.CallOptions{
			Model:        model,
			SystemPrompt: sysPrompt,
			Tools:        availTools,
			MaxTokens:    8192,
		}, history)
		if err != nil {
			return finalText.String(), fmt.Errorf("spawn: %w", err)
		}

		var assistantBlocks []core.Block
		pending := map[int]*pendingBlock{}
		var stopReason string

		for ev := range ch {
			switch e := ev.(type) {
			case core.TextDeltaEvent:
				pb := pending[e.Index]
				if pb == nil {
					pb = &pendingBlock{kind: core.KindText}
					pending[e.Index] = pb
				}
				pb.text.WriteString(e.Text)
			case core.ToolUseStartEvent:
				pending[e.Index] = &pendingBlock{kind: core.KindToolUse, toolID: e.ID, toolName: e.Name}
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
			case core.ErrorEvent:
				return finalText.String(), e.Err
			}
		}

		for _, b := range assistantBlocks {
			if t, ok := b.(core.TextBlock); ok {
				finalText.WriteString(t.Text)
				finalText.WriteString("\n")
			}
		}

		history = append(history, core.Message{Role: core.RoleAssistant, Blocks: assistantBlocks})

		var toolUses []core.ToolUseBlock
		for _, b := range assistantBlocks {
			if tu, ok := b.(core.ToolUseBlock); ok {
				toolUses = append(toolUses, tu)
			}
		}

		if len(toolUses) == 0 || stopReason != "tool_use" {
			return strings.TrimSpace(finalText.String()), nil
		}

		var resultBlocks []core.Block
		for _, tu := range toolUses {
			output, isErr := s.executeTool(ctx, tu, availTools)
			resultBlocks = append(resultBlocks, core.ToolResultBlock{
				UseID:   tu.ID,
				Content: output,
				IsError: isErr,
			})
		}
		history = append(history, core.Message{Role: core.RoleUser, Blocks: resultBlocks})
	}

	return strings.TrimSpace(finalText.String()), errors.New("max sub-agent turns exceeded")
}

func (s *spawner) executeTool(ctx context.Context, tu core.ToolUseBlock, avail []core.Tool) (string, bool) {
	var tool core.Tool
	for _, t := range avail {
		if t.Name() == tu.Name {
			tool = t
			break
		}
	}
	if tool == nil {
		return fmt.Sprintf("tool not found: %s", tu.Name), true
	}
	if s.cfg.Permissions != nil {
		decision, reason := s.cfg.Permissions.Check(ctx, core.PermissionRequest{
			Tool:  tu.Name,
			Input: []byte(tu.Input),
		})
		switch decision {
		case core.PermissionDeny:
			return fmt.Sprintf("permission denied: %s", reason), true
		case core.PermissionAsk:
			return fmt.Sprintf("permission would prompt; sub-agent auto-denied: %s", reason), true
		}
	}
	output, err := tool.Run(ctx, tu.Input)
	if err != nil {
		return fmt.Sprintf("error: %v", err), true
	}
	return output, false
}

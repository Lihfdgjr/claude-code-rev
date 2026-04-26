package chat

import (
	"strings"

	"claudecode/internal/core"
)

// ContextLimit returns the approximate input context window (in tokens) for
// the given model id.
func ContextLimit(model string) int {
	switch {
	case strings.HasPrefix(model, "claude-opus-"):
		return 200000
	case strings.HasPrefix(model, "claude-sonnet-"):
		return 200000
	case strings.HasPrefix(model, "claude-haiku-"):
		return 200000
	default:
		return 100000
	}
}

// Usage estimates the number of tokens currently held in history. It is a
// rough heuristic: characters-per-block divided by four, with fixed
// overhead for tool_use blocks.
func Usage(history []core.Message) int {
	chars := 0
	for _, m := range history {
		for _, b := range m.Blocks {
			switch v := b.(type) {
			case core.TextBlock:
				chars += len(v.Text)
			case core.ThinkingBlock:
				chars += len(v.Text)
			case core.ToolUseBlock:
				chars += 100 + len(v.Input)
			case core.ToolResultBlock:
				chars += len(v.Content)
			}
		}
	}
	return chars / 4
}

// BudgetPercent returns the fraction of the model's context window used by
// history. 1.0 means full.
func BudgetPercent(history []core.Message, model string) float64 {
	limit := ContextLimit(model)
	if limit <= 0 {
		return 0
	}
	return float64(Usage(history)) / float64(limit)
}

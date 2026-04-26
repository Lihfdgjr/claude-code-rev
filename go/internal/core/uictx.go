package core

import "context"

type ctxKey struct{ name string }

var (
	uiEventsKey      = ctxKey{"ui-events"}
	subagentDepthKey = ctxKey{"subagent-depth"}
)

// MaxSubagentDepth caps how deeply sub-agents may recursively spawn
// further sub-agents.
const MaxSubagentDepth = 3

// WithUIEvents attaches a UI event channel to a context so tools running
// during a turn can push interactive events (UIAskUserEvent, etc.).
func WithUIEvents(ctx context.Context, ch chan<- UIEvent) context.Context {
	return context.WithValue(ctx, uiEventsKey, ch)
}

// UIEvents returns the UI event channel attached to ctx, or nil if none.
func UIEvents(ctx context.Context) chan<- UIEvent {
	v := ctx.Value(uiEventsKey)
	if v == nil {
		return nil
	}
	if ch, ok := v.(chan<- UIEvent); ok {
		return ch
	}
	return nil
}

// WithSubagentDepth records the current sub-agent recursion depth on ctx.
func WithSubagentDepth(ctx context.Context, d int) context.Context {
	return context.WithValue(ctx, subagentDepthKey, d)
}

// SubagentDepth returns the recursion depth attached to ctx, or 0 if none.
func SubagentDepth(ctx context.Context) int {
	v := ctx.Value(subagentDepthKey)
	if v == nil {
		return 0
	}
	if d, ok := v.(int); ok {
		return d
	}
	return 0
}

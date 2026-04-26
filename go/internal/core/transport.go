package core

import "context"

type CallOptions struct {
	Model        string
	SystemPrompt string
	Tools        []Tool
	MaxTokens    int
	Temperature  *float64
	Thinking     bool
}

type StreamEventKind string

const (
	EventMessageStart  StreamEventKind = "message_start"
	EventTextDelta     StreamEventKind = "text_delta"
	EventThinkingDelta StreamEventKind = "thinking_delta"
	EventToolUseStart  StreamEventKind = "tool_use_start"
	EventToolInputDelta StreamEventKind = "tool_input_delta"
	EventBlockEnd      StreamEventKind = "block_end"
	EventMessageEnd    StreamEventKind = "message_end"
	EventError         StreamEventKind = "error"
)

type StreamEvent interface {
	EventKind() StreamEventKind
}

type MessageStartEvent struct {
	ID    string
	Model string
}

func (MessageStartEvent) EventKind() StreamEventKind { return EventMessageStart }

type TextDeltaEvent struct {
	Index int
	Text  string
}

func (TextDeltaEvent) EventKind() StreamEventKind { return EventTextDelta }

type ThinkingDeltaEvent struct {
	Index int
	Text  string
}

func (ThinkingDeltaEvent) EventKind() StreamEventKind { return EventThinkingDelta }

type ToolUseStartEvent struct {
	Index int
	ID    string
	Name  string
}

func (ToolUseStartEvent) EventKind() StreamEventKind { return EventToolUseStart }

type ToolInputDeltaEvent struct {
	Index   int
	JSONPart string
}

func (ToolInputDeltaEvent) EventKind() StreamEventKind { return EventToolInputDelta }

type BlockEndEvent struct {
	Index int
}

func (BlockEndEvent) EventKind() StreamEventKind { return EventBlockEnd }

type MessageEndEvent struct {
	StopReason string
	Usage      Usage
}

func (MessageEndEvent) EventKind() StreamEventKind { return EventMessageEnd }

type ErrorEvent struct {
	Err error
}

func (ErrorEvent) EventKind() StreamEventKind { return EventError }

type Transport interface {
	Stream(ctx context.Context, opts CallOptions, history []Message) (<-chan StreamEvent, error)
}

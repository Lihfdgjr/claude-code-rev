package core

import "context"

type NotifyLevel int

const (
	NotifyInfo NotifyLevel = iota
	NotifyWarn
	NotifyError
	NotifyDebug
)

type Session interface {
	History() []Message
	ResetHistory()
	Append(m Message)

	// Snapshot returns a read-only copy of the current history.
	Snapshot() []Message
	// Restore replaces the history with the given slice.
	Restore(messages []Message)
	// Checkpoint records the current history into the undo stack.
	Checkpoint(label string)
	// Undo pops the latest checkpoint and restores its history; the
	// pre-undo history is pushed onto the redo stack.
	Undo() (label string, ok bool)
	// Redo is the inverse of Undo.
	Redo() (label string, ok bool)

	Model() string
	SetModel(id string)

	SystemPrompt() string
	SetSystemPrompt(s string)

	Notify(level NotifyLevel, msg string)
	Compact(ctx context.Context) error

	CumulativeUsage() Usage
	AddUsage(u Usage)

	// Attach queues an attachment (image/audio/document block) to be
	// prepended to the next user message. DrainAttachments returns and
	// clears the queue (called by Driver.Submit).
	Attach(b Block)
	DrainAttachments() []Block

	// Title returns the session title (auto-set after first turn) and
	// SetTitle overrides it explicitly.
	Title() string
	SetTitle(t string)

	// Resubmit re-sends text as a new user turn (used by /retry).
	Resubmit(text string)
	// Cancel aborts any in-flight turn (used by /cancel).
	Cancel()
}

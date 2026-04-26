package chat

import (
	"context"
	"strings"
	"sync"

	"claudecode/internal/core"
)

type checkpoint struct {
	Label   string
	History []core.Message
}

const maxCheckpointStack = 20

type session struct {
	mu           sync.Mutex
	history      []core.Message
	systemPrompt string
	model        string
	transport    core.Transport
	notifier     func(level core.NotifyLevel, msg string)
	cumUsage     core.Usage
	attachments  []core.Block
	title        string
	resubmit     func(text string)
	cancel       func()
	undoStack    []checkpoint
	redoStack    []checkpoint
}

func (s *session) History() []core.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]core.Message, len(s.history))
	copy(out, s.history)
	return out
}

func (s *session) ResetHistory() {
	s.mu.Lock()
	s.history = nil
	s.mu.Unlock()
}

func (s *session) Append(m core.Message) {
	s.mu.Lock()
	s.history = append(s.history, m)
	s.mu.Unlock()
}

func (s *session) Model() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.model
}

func (s *session) SetModel(id string) {
	s.mu.Lock()
	s.model = id
	s.mu.Unlock()
}

func (s *session) SystemPrompt() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.systemPrompt
}

func (s *session) SetSystemPrompt(p string) {
	s.mu.Lock()
	s.systemPrompt = p
	s.mu.Unlock()
}

func (s *session) Notify(level core.NotifyLevel, msg string) {
	if s.notifier != nil {
		s.notifier(level, msg)
	}
}

func (s *session) CumulativeUsage() core.Usage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cumUsage
}

func (s *session) AddUsage(u core.Usage) {
	s.mu.Lock()
	s.cumUsage.InputTokens += u.InputTokens
	s.cumUsage.OutputTokens += u.OutputTokens
	s.cumUsage.CacheReadTokens += u.CacheReadTokens
	s.cumUsage.CacheCreationTokens += u.CacheCreationTokens
	s.mu.Unlock()
}

func (s *session) Attach(b core.Block) {
	s.mu.Lock()
	s.attachments = append(s.attachments, b)
	s.mu.Unlock()
}

func (s *session) DrainAttachments() []core.Block {
	s.mu.Lock()
	out := s.attachments
	s.attachments = nil
	s.mu.Unlock()
	return out
}

func (s *session) Title() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.title
}

func (s *session) SetTitle(t string) {
	s.mu.Lock()
	s.title = t
	s.mu.Unlock()
}

func (s *session) Resubmit(text string) {
	s.mu.Lock()
	fn := s.resubmit
	s.mu.Unlock()
	if fn != nil {
		fn(text)
	}
}

func (s *session) Cancel() {
	s.mu.Lock()
	fn := s.cancel
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func (s *session) Snapshot() []core.Message {
	return s.History()
}

func (s *session) Restore(messages []core.Message) {
	cp := append([]core.Message(nil), messages...)
	s.mu.Lock()
	s.history = cp
	s.mu.Unlock()
}

func (s *session) Checkpoint(label string) {
	s.mu.Lock()
	cp := checkpoint{Label: label, History: append([]core.Message(nil), s.history...)}
	s.undoStack = append(s.undoStack, cp)
	if len(s.undoStack) > maxCheckpointStack {
		s.undoStack = s.undoStack[len(s.undoStack)-maxCheckpointStack:]
	}
	s.redoStack = nil
	s.mu.Unlock()
}

func (s *session) Undo() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.undoStack)
	if n == 0 {
		return "", false
	}
	top := s.undoStack[n-1]
	s.undoStack = s.undoStack[:n-1]
	current := checkpoint{Label: top.Label, History: append([]core.Message(nil), s.history...)}
	s.redoStack = append(s.redoStack, current)
	if len(s.redoStack) > maxCheckpointStack {
		s.redoStack = s.redoStack[len(s.redoStack)-maxCheckpointStack:]
	}
	s.history = append([]core.Message(nil), top.History...)
	return top.Label, true
}

func (s *session) Redo() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.redoStack)
	if n == 0 {
		return "", false
	}
	top := s.redoStack[n-1]
	s.redoStack = s.redoStack[:n-1]
	current := checkpoint{Label: top.Label, History: append([]core.Message(nil), s.history...)}
	s.undoStack = append(s.undoStack, current)
	if len(s.undoStack) > maxCheckpointStack {
		s.undoStack = s.undoStack[len(s.undoStack)-maxCheckpointStack:]
	}
	s.history = append([]core.Message(nil), top.History...)
	return top.Label, true
}

func (s *session) Compact(ctx context.Context) error {
	s.mu.Lock()
	if len(s.history) == 0 {
		s.mu.Unlock()
		return nil
	}
	h := append([]core.Message(nil), s.history...)
	sys := s.systemPrompt
	model := s.model
	s.mu.Unlock()

	h = append(h, core.Message{
		Role: core.RoleUser,
		Blocks: []core.Block{core.TextBlock{
			Text: "Summarize the conversation above in roughly 200 words. Capture key decisions, code changes, and unresolved questions. Output only the summary.",
		}},
	})

	ch, err := s.transport.Stream(ctx, core.CallOptions{
		Model:        model,
		SystemPrompt: sys,
		MaxTokens:    1024,
	}, h)
	if err != nil {
		return err
	}

	var sb strings.Builder
	for ev := range ch {
		switch e := ev.(type) {
		case core.TextDeltaEvent:
			sb.WriteString(e.Text)
		case core.ErrorEvent:
			return e.Err
		}
	}

	summary := strings.TrimSpace(sb.String())
	if summary == "" {
		summary = "(empty summary)"
	}

	s.mu.Lock()
	s.history = []core.Message{
		{Role: core.RoleUser, Blocks: []core.Block{core.TextBlock{
			Text: "Summary of prior conversation:\n\n" + summary,
		}}},
	}
	s.mu.Unlock()
	return nil
}

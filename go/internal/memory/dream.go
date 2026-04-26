package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"claudecode/internal/core"
)

const dreamSystemPrompt = `You distill a conversation transcript into durable memory entries for a coding assistant.
Read the transcript and extract NEW, NON-OBVIOUS facts worth persisting.

Output STRICT JSON only:
{"entries":[{"name":"slug-style-name","type":"user|feedback|project|reference","description":"one-line hook","body":"markdown body 1-3 short paragraphs"}]}

Rules:
- ONLY include items the assistant did not already know.
- Skip ephemeral task details, code snippets, and anything obvious from the codebase.
- "user" = role/preferences/expertise/constraints
- "feedback" = corrections or validated approaches the user gave (include WHY and WHEN)
- "project" = state of work, deadlines, decisions
- "reference" = external systems / dashboards / docs locations
- If nothing is worth saving, return {"entries":[]}.
- Output ONLY the JSON. No prose, no fences.`

// AutoDream calls the model to extract memory entries from history and saves
// any new ones to the store. Returns the number of entries saved.
func AutoDream(ctx context.Context, store *Store, transport core.Transport, model string, history []core.Message) (int, error) {
	if store == nil || transport == nil {
		return 0, nil
	}
	if len(history) < 2 {
		return 0, nil
	}

	transcript := buildTranscript(history)
	if transcript == "" {
		return 0, nil
	}

	prompt := []core.Message{
		{Role: core.RoleUser, Blocks: []core.Block{core.TextBlock{Text: "Conversation transcript follows:\n\n" + transcript}}},
	}

	ch, err := transport.Stream(ctx, core.CallOptions{
		Model:        model,
		SystemPrompt: dreamSystemPrompt,
		MaxTokens:    1200,
	}, prompt)
	if err != nil {
		return 0, err
	}

	var sb strings.Builder
	for ev := range ch {
		switch e := ev.(type) {
		case core.TextDeltaEvent:
			sb.WriteString(e.Text)
		case core.ErrorEvent:
			return 0, e.Err
		}
	}

	out := strings.TrimSpace(sb.String())
	out = stripJSONFences(out)

	var parsed struct {
		Entries []struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Description string `json:"description"`
			Body        string `json:"body"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return 0, fmt.Errorf("dream: parse JSON: %w (raw=%q)", err, truncate(out, 200))
	}

	saved := 0
	for _, e := range parsed.Entries {
		if e.Name == "" || e.Body == "" {
			continue
		}
		ent := &Entry{
			Name:        e.Name,
			Type:        e.Type,
			Description: e.Description,
			Body:        e.Body,
		}
		if existing, _ := store.Get(e.Name); existing != nil {
			// Skip duplicates (same body) — otherwise overwrite.
			if strings.TrimSpace(existing.Body) == strings.TrimSpace(e.Body) {
				continue
			}
		}
		if err := store.Save(ent); err == nil {
			saved++
		}
	}
	return saved, nil
}

func buildTranscript(history []core.Message) string {
	var b strings.Builder
	for _, m := range history {
		role := strings.ToUpper(string(m.Role))
		for _, blk := range m.Blocks {
			switch v := blk.(type) {
			case core.TextBlock:
				if t := strings.TrimSpace(v.Text); t != "" {
					fmt.Fprintf(&b, "[%s] %s\n", role, t)
				}
			case core.ToolUseBlock:
				fmt.Fprintf(&b, "[%s tool] %s\n", role, v.Name)
			case core.ToolResultBlock:
				snippet := strings.TrimSpace(v.Content)
				if len(snippet) > 200 {
					snippet = snippet[:200] + "..."
				}
				fmt.Fprintf(&b, "[%s result] %s\n", role, snippet)
			}
		}
	}
	return b.String()
}

func stripJSONFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// drop opening fence
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		// drop closing fence
		if j := strings.LastIndex(s, "```"); j >= 0 {
			s = s[:j]
		}
	}
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// DreamScheduler debounces auto-dream invocations: at most one call in flight
// at a time, with a configurable minimum interval between successful runs.
type DreamScheduler struct {
	mu        sync.Mutex
	inFlight  bool
	last      time.Time
	MinPeriod time.Duration
	Store     *Store
	Transport core.Transport
	ModelOf   func() string
	HistoryOf func() []core.Message
	OnDone    func(saved int, err error)
}

func (s *DreamScheduler) MaybeRun() {
	if s == nil || s.Store == nil || s.Transport == nil {
		return
	}
	period := s.MinPeriod
	if period <= 0 {
		period = 2 * time.Minute
	}
	s.mu.Lock()
	if s.inFlight || time.Since(s.last) < period {
		s.mu.Unlock()
		return
	}
	s.inFlight = true
	s.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		model := "claude-haiku-4-5-20251001"
		if s.ModelOf != nil {
			if m := s.ModelOf(); m != "" {
				model = m
			}
		}
		var hist []core.Message
		if s.HistoryOf != nil {
			hist = s.HistoryOf()
		}
		saved, err := AutoDream(ctx, s.Store, s.Transport, model, hist)
		s.mu.Lock()
		s.inFlight = false
		s.last = time.Now()
		s.mu.Unlock()
		if s.OnDone != nil {
			s.OnDone(saved, err)
		}
	}()
}

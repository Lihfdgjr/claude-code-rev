package sessions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"claudecode/internal/core"
)

// Store persists session snapshots as JSON files under a root directory.
type Store struct {
	root string
}

// New creates a Store rooted at rootDir, creating the directory if missing.
func New(rootDir string) *Store {
	_ = os.MkdirAll(rootDir, 0o755)
	return &Store{root: rootDir}
}

// Snapshot is the serialized form of a session.
type Snapshot struct {
	ID           string            `json:"id"`
	CreatedAt    time.Time         `json:"created_at"`
	LastModified time.Time         `json:"last_modified"`
	Model        string            `json:"model"`
	SystemPrompt string            `json:"system_prompt"`
	Summary      string            `json:"summary"`
	Messages     []SerializedMsg   `json:"messages"`
}

// Meta is the lightweight summary used for listings.
type Meta struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	LastModified time.Time `json:"last_modified"`
	MessageCount int       `json:"message_count"`
	Model        string    `json:"model"`
	Summary      string    `json:"summary"`
}

// SerializedMsg is a JSON-friendly message with tagged-union blocks.
type SerializedMsg struct {
	Role   string            `json:"role"`
	Blocks []json.RawMessage `json:"blocks"`
}

// Tagged-union block representations.
type textBlockJSON struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type toolUseBlockJSON struct {
	Kind  string          `json:"kind"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type toolResultBlockJSON struct {
	Kind    string `json:"kind"`
	UseID   string `json:"use_id"`
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

type thinkingBlockJSON struct {
	Kind      string `json:"kind"`
	Text      string `json:"text"`
	Signature string `json:"signature"`
}

// Save writes the snapshot atomically as <root>/<id>.json.
func (s *Store) Save(id string, sess Snapshot) error {
	if id == "" {
		return errors.New("sessions: empty id")
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}
	if sess.ID == "" {
		sess.ID = id
	}
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = time.Now().UTC()
	}
	sess.LastModified = time.Now().UTC()

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	final := filepath.Join(s.root, id+".json")
	tmp := final + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Load reads a session snapshot by id.
func (s *Store) Load(id string) (*Snapshot, error) {
	if id == "" {
		return nil, errors.New("sessions: empty id")
	}
	path := filepath.Join(s.root, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("sessions: parse %s: %w", id, err)
	}
	return &snap, nil
}

// List returns metadata for every session, sorted by last_modified desc.
func (s *Store) List() ([]*Meta, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]*Meta, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(s.root, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Decode just the metadata fields, skipping messages content.
		var head struct {
			ID           string          `json:"id"`
			CreatedAt    time.Time       `json:"created_at"`
			LastModified time.Time       `json:"last_modified"`
			Model        string          `json:"model"`
			Summary      string          `json:"summary"`
			Messages     []SerializedMsg `json:"messages"`
		}
		if err := json.Unmarshal(data, &head); err != nil {
			continue
		}
		id := head.ID
		if id == "" {
			id = strings.TrimSuffix(name, ".json")
		}
		out = append(out, &Meta{
			ID:           id,
			CreatedAt:    head.CreatedAt,
			LastModified: head.LastModified,
			MessageCount: len(head.Messages),
			Model:        head.Model,
			Summary:      head.Summary,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastModified.After(out[j].LastModified)
	})
	return out, nil
}

// Delete removes the session file by id.
func (s *Store) Delete(id string) error {
	if id == "" {
		return errors.New("sessions: empty id")
	}
	return os.Remove(filepath.Join(s.root, id+".json"))
}

// SnapshotFromSession builds a Snapshot from a live core.Session.
func SnapshotFromSession(id string, sess core.Session) Snapshot {
	msgs := sess.History()
	return Snapshot{
		ID:           id,
		CreatedAt:    time.Now().UTC(),
		LastModified: time.Now().UTC(),
		Model:        sess.Model(),
		SystemPrompt: sess.SystemPrompt(),
		Summary:      summaryOf(msgs),
		Messages:     SerializeMessages(msgs),
	}
}

// SerializeMessages converts core.Message values to JSON-friendly form.
func SerializeMessages(msgs []core.Message) []SerializedMsg {
	out := make([]SerializedMsg, 0, len(msgs))
	for _, m := range msgs {
		sm := SerializedMsg{Role: string(m.Role)}
		for _, b := range m.Blocks {
			raw, err := encodeBlock(b)
			if err != nil || raw == nil {
				continue
			}
			sm.Blocks = append(sm.Blocks, raw)
		}
		out = append(out, sm)
	}
	return out
}

// DeserializeMessages converts SerializedMsg values back to core.Message.
func DeserializeMessages(in []SerializedMsg) ([]core.Message, error) {
	out := make([]core.Message, 0, len(in))
	for _, sm := range in {
		msg := core.Message{Role: core.Role(sm.Role)}
		for _, raw := range sm.Blocks {
			b, err := decodeBlock(raw)
			if err != nil {
				return nil, err
			}
			if b != nil {
				msg.Blocks = append(msg.Blocks, b)
			}
		}
		out = append(out, msg)
	}
	return out, nil
}

func encodeBlock(b core.Block) (json.RawMessage, error) {
	switch v := b.(type) {
	case core.TextBlock:
		return json.Marshal(textBlockJSON{Kind: string(core.KindText), Text: v.Text})
	case core.ToolUseBlock:
		input := v.Input
		if len(input) == 0 {
			input = json.RawMessage("null")
		}
		return json.Marshal(toolUseBlockJSON{
			Kind:  string(core.KindToolUse),
			ID:    v.ID,
			Name:  v.Name,
			Input: input,
		})
	case core.ToolResultBlock:
		return json.Marshal(toolResultBlockJSON{
			Kind:    string(core.KindToolResult),
			UseID:   v.UseID,
			Content: v.Content,
			IsError: v.IsError,
		})
	case core.ThinkingBlock:
		return json.Marshal(thinkingBlockJSON{
			Kind:      string(core.KindThinking),
			Text:      v.Text,
			Signature: v.Signature,
		})
	default:
		return nil, nil
	}
}

func decodeBlock(raw json.RawMessage) (core.Block, error) {
	var head struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	switch core.BlockKind(head.Kind) {
	case core.KindText:
		var t textBlockJSON
		if err := json.Unmarshal(raw, &t); err != nil {
			return nil, err
		}
		return core.TextBlock{Text: t.Text}, nil
	case core.KindToolUse:
		var t toolUseBlockJSON
		if err := json.Unmarshal(raw, &t); err != nil {
			return nil, err
		}
		return core.ToolUseBlock{ID: t.ID, Name: t.Name, Input: t.Input}, nil
	case core.KindToolResult:
		var t toolResultBlockJSON
		if err := json.Unmarshal(raw, &t); err != nil {
			return nil, err
		}
		return core.ToolResultBlock{UseID: t.UseID, Content: t.Content, IsError: t.IsError}, nil
	case core.KindThinking:
		var t thinkingBlockJSON
		if err := json.Unmarshal(raw, &t); err != nil {
			return nil, err
		}
		return core.ThinkingBlock{Text: t.Text, Signature: t.Signature}, nil
	default:
		return nil, fmt.Errorf("sessions: unknown block kind %q", head.Kind)
	}
}

// summaryOf returns the first user message text, truncated.
func summaryOf(msgs []core.Message) string {
	const maxLen = 80
	for _, m := range msgs {
		if m.Role != core.RoleUser {
			continue
		}
		for _, b := range m.Blocks {
			if t, ok := b.(core.TextBlock); ok {
				s := strings.TrimSpace(t.Text)
				if s == "" {
					continue
				}
				if len(s) > maxLen {
					s = s[:maxLen] + "..."
				}
				return s
			}
		}
	}
	return ""
}

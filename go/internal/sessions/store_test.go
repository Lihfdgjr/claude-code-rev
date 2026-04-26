package sessions

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"claudecode/internal/core"
)

func TestSaveLoadRoundTripPreservesAllBlockKinds(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	msgs := []core.Message{
		{
			Role: core.RoleUser,
			Blocks: []core.Block{
				core.TextBlock{Text: "hello"},
			},
		},
		{
			Role: core.RoleAssistant,
			Blocks: []core.Block{
				core.ThinkingBlock{Text: "thinking", Signature: "sig"},
				core.ToolUseBlock{ID: "u1", Name: "Calc", Input: json.RawMessage(`{"a":1}`)},
			},
		},
		{
			Role: core.RoleUser,
			Blocks: []core.Block{
				core.ToolResultBlock{UseID: "u1", Content: "ok", IsError: false},
			},
		},
	}

	snap := Snapshot{
		ID:           "session1",
		Model:        "claude-test",
		SystemPrompt: "you are helpful",
		Summary:      "test",
		Messages:     SerializeMessages(msgs),
	}
	if err := s.Save("session1", snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := s.Load("session1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ID != "session1" || loaded.Model != "claude-test" || loaded.SystemPrompt != "you are helpful" {
		t.Errorf("metadata mismatch: %+v", loaded)
	}
	if loaded.LastModified.IsZero() {
		t.Error("LastModified should be set")
	}

	got, err := DeserializeMessages(loaded.Messages)
	if err != nil {
		t.Fatalf("DeserializeMessages: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(got))
	}

	if got[0].Role != core.RoleUser || len(got[0].Blocks) != 1 {
		t.Fatalf("msg0 mismatch: %+v", got[0])
	}
	tb, ok := got[0].Blocks[0].(core.TextBlock)
	if !ok || tb.Text != "hello" {
		t.Errorf("text block round-trip = %+v (ok=%v)", got[0].Blocks[0], ok)
	}

	if len(got[1].Blocks) != 2 {
		t.Fatalf("msg1 expected 2 blocks, got %d", len(got[1].Blocks))
	}
	thb, ok := got[1].Blocks[0].(core.ThinkingBlock)
	if !ok || thb.Text != "thinking" || thb.Signature != "sig" {
		t.Errorf("thinking block round-trip = %+v (ok=%v)", got[1].Blocks[0], ok)
	}
	tu, ok := got[1].Blocks[1].(core.ToolUseBlock)
	if !ok || tu.ID != "u1" || tu.Name != "Calc" {
		t.Errorf("tool_use block round-trip = %+v (ok=%v)", got[1].Blocks[1], ok)
	} else {
		var compact bytes.Buffer
		if err := json.Compact(&compact, tu.Input); err != nil {
			t.Errorf("compact tool_use input: %v", err)
		} else if compact.String() != `{"a":1}` {
			t.Errorf("tool_use input round-trip = %q, want %q", compact.String(), `{"a":1}`)
		}
	}

	tr, ok := got[2].Blocks[0].(core.ToolResultBlock)
	if !ok || tr.UseID != "u1" || tr.Content != "ok" || tr.IsError {
		t.Errorf("tool_result block round-trip = %+v (ok=%v)", got[2].Blocks[0], ok)
	}
}

func TestListSortsByLastModifiedDesc(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	for _, id := range []string{"a", "b", "c"} {
		if err := s.Save(id, Snapshot{ID: id}); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
		// Ensure distinct LastModified timestamps.
		time.Sleep(10 * time.Millisecond)
	}

	metas, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 3 {
		t.Fatalf("expected 3 metas, got %d", len(metas))
	}

	// "c" was saved last, so it must be first.
	if metas[0].ID != "c" {
		t.Errorf("first = %q, want \"c\"", metas[0].ID)
	}
	if metas[2].ID != "a" {
		t.Errorf("last = %q, want \"a\"", metas[2].ID)
	}

	// Verify ordering by timestamp.
	for i := 0; i+1 < len(metas); i++ {
		if metas[i].LastModified.Before(metas[i+1].LastModified) {
			t.Errorf("metas not sorted desc at %d: %v < %v", i, metas[i].LastModified, metas[i+1].LastModified)
		}
	}
}

func TestSaveEmptyIDErrors(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Save("", Snapshot{}); err == nil {
		t.Error("expected error for empty id")
	}
}

func TestLoadMissingErrors(t *testing.T) {
	s := New(t.TempDir())
	if _, err := s.Load("does-not-exist"); err == nil {
		t.Error("expected error for missing id")
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Save("toremove", Snapshot{ID: "toremove"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Delete("toremove"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Load("toremove"); err == nil {
		t.Error("expected error after delete")
	}
}

func TestListEmptyDir(t *testing.T) {
	s := New(t.TempDir())
	metas, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 0 {
		t.Errorf("expected 0 metas, got %d", len(metas))
	}
}

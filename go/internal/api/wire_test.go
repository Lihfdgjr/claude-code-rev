package api

import (
	"encoding/json"
	"testing"

	"claudecode/internal/core"
)

func TestMessagesFromCoreRoundTripsAllBlockKinds(t *testing.T) {
	msgs := []core.Message{
		{
			Role: core.RoleUser,
			Blocks: []core.Block{
				core.TextBlock{Text: "hello"},
				core.ImageBlock{Source: "AAAA", MediaType: "image/png"},
			},
		},
		{
			Role: core.RoleAssistant,
			Blocks: []core.Block{
				core.ThinkingBlock{Text: "ponder", Signature: "sig"},
				core.ToolUseBlock{ID: "tool1", Name: "Calc", Input: json.RawMessage(`{"a":1}`)},
			},
		},
		{
			Role: core.RoleUser,
			Blocks: []core.Block{
				core.ToolResultBlock{UseID: "tool1", Content: "42", IsError: false},
			},
		},
	}

	out := messagesFromCore(msgs)
	if len(out) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(out))
	}

	// First message: user with text + image.
	if out[0].Role != "user" {
		t.Errorf("msg0 role = %q, want \"user\"", out[0].Role)
	}
	if len(out[0].Content) != 2 {
		t.Fatalf("msg0 expected 2 blocks, got %d", len(out[0].Content))
	}
	if out[0].Content[0].Type != "text" || out[0].Content[0].Text != "hello" {
		t.Errorf("msg0 block0 = %+v", out[0].Content[0])
	}
	if out[0].Content[1].Type != "image" {
		t.Errorf("msg0 block1.Type = %q, want \"image\"", out[0].Content[1].Type)
	}
	if out[0].Content[1].Source == nil {
		t.Fatal("msg0 block1.Source is nil")
	}
	if out[0].Content[1].Source.MediaType != "image/png" {
		t.Errorf("image MediaType = %q", out[0].Content[1].Source.MediaType)
	}
	if out[0].Content[1].Source.Data != "AAAA" {
		t.Errorf("image Data = %q", out[0].Content[1].Source.Data)
	}
	if out[0].Content[1].Source.Type != "base64" {
		t.Errorf("image Source.Type = %q", out[0].Content[1].Source.Type)
	}

	// Second message: assistant with thinking + tool_use.
	if out[1].Content[0].Type != "thinking" || out[1].Content[0].Thinking != "ponder" || out[1].Content[0].Signature != "sig" {
		t.Errorf("thinking block = %+v", out[1].Content[0])
	}
	if out[1].Content[1].Type != "tool_use" || out[1].Content[1].ID != "tool1" || out[1].Content[1].Name != "Calc" {
		t.Errorf("tool_use block = %+v", out[1].Content[1])
	}
	if string(out[1].Content[1].Input) != `{"a":1}` {
		t.Errorf("tool_use Input = %s", string(out[1].Content[1].Input))
	}

	// Third message: tool_result.
	if out[2].Content[0].Type != "tool_result" || out[2].Content[0].ToolUseID != "tool1" || out[2].Content[0].Content != "42" {
		t.Errorf("tool_result block = %+v", out[2].Content[0])
	}
}

func TestMessagesFromCoreEmptyToolUseInputDefaultsToObject(t *testing.T) {
	msgs := []core.Message{{
		Role:   core.RoleAssistant,
		Blocks: []core.Block{core.ToolUseBlock{ID: "x", Name: "n"}},
	}}
	out := messagesFromCore(msgs)
	if len(out) != 1 || len(out[0].Content) != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
	if string(out[0].Content[0].Input) != "{}" {
		t.Errorf("empty input default = %s", string(out[0].Content[0].Input))
	}
}

func TestMessagesFromCoreSkipsEmptyMessages(t *testing.T) {
	msgs := []core.Message{
		{Role: core.RoleUser, Blocks: nil},
		{Role: core.RoleUser, Blocks: []core.Block{core.TextBlock{Text: "x"}}},
	}
	out := messagesFromCore(msgs)
	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}
}

func TestBlockTextMarshalRoundTrip(t *testing.T) {
	b := Block{Type: "text", Text: "hello"}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Block
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != "text" || got.Text != "hello" {
		t.Errorf("round-trip = %+v", got)
	}
}

func TestBlockToolUseMarshalRoundTrip(t *testing.T) {
	b := Block{Type: "tool_use", ID: "id", Name: "Calc", Input: json.RawMessage(`{"x":1}`)}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Block
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != "tool_use" || got.ID != "id" || got.Name != "Calc" || string(got.Input) != `{"x":1}` {
		t.Errorf("round-trip = %+v", got)
	}
}

func TestBlockImageMarshalRoundTrip(t *testing.T) {
	b := Block{Type: "image", Source: &BlockSource{Type: "base64", MediaType: "image/png", Data: "AAA"}}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Block
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != "image" || got.Source == nil || got.Source.Data != "AAA" {
		t.Errorf("round-trip = %+v", got)
	}
}

func TestBlockToolResultMarshalRoundTrip(t *testing.T) {
	b := Block{Type: "tool_result", ToolUseID: "u1", Content: "ok", IsError: true}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Block
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != "tool_result" || got.ToolUseID != "u1" || got.Content != "ok" || !got.IsError {
		t.Errorf("round-trip = %+v", got)
	}
}

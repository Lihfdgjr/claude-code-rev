package core

import "testing"

func TestBlockKinds(t *testing.T) {
	cases := []struct {
		name  string
		block Block
		want  BlockKind
	}{
		{"text", TextBlock{Text: "hi"}, KindText},
		{"tool_use", ToolUseBlock{ID: "id", Name: "n"}, KindToolUse},
		{"tool_result", ToolResultBlock{UseID: "id"}, KindToolResult},
		{"thinking", ThinkingBlock{Text: "t"}, KindThinking},
		{"image", ImageBlock{Source: "s", MediaType: "image/png"}, KindImage},
		{"audio", AudioBlock{Source: "s", MediaType: "audio/wav"}, KindAudio},
		{"document", DocumentBlock{Source: "s", MediaType: "application/pdf"}, KindDocument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.block.Kind(); got != tc.want {
				t.Fatalf("Kind() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRoleConstants(t *testing.T) {
	if RoleUser != "user" {
		t.Errorf("RoleUser = %q, want \"user\"", RoleUser)
	}
	if RoleAssistant != "assistant" {
		t.Errorf("RoleAssistant = %q, want \"assistant\"", RoleAssistant)
	}
	if RoleSystem != "system" {
		t.Errorf("RoleSystem = %q, want \"system\"", RoleSystem)
	}
}

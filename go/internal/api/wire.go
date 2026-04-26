package api

import (
	"encoding/json"

	"claudecode/internal/core"
)

// Wire types for the Anthropic Messages API request/response JSON.

type Request struct {
	Model       string    `json:"model"`
	System      string    `json:"system,omitempty"`
	Messages    []Message `json:"messages"`
	Tools       []ToolDef `json:"tools,omitempty"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature *float64  `json:"temperature,omitempty"`
	Stream      bool      `json:"stream"`
	Thinking    *Thinking `json:"thinking,omitempty"`
}

type Thinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

type CacheControl struct {
	Type string `json:"type"`
}

type Message struct {
	Role    string  `json:"role"`
	Content []Block `json:"content"`
}

// Block is a polymorphic content block. Marshalling is custom; unmarshalling
// uses the Type field plus the typed fields populated by the JSON.
type Block struct {
	Type string `json:"type"`

	// text
	Text      string     `json:"text,omitempty"`
	Citations []Citation `json:"citations,omitempty"`

	// tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`

	// thinking
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`

	// image / audio / document
	Source *BlockSource `json:"source,omitempty"`
	Title  string       `json:"title,omitempty"`

	// caching
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// BlockSource carries base64-encoded media for image/audio/document blocks.
type BlockSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // e.g. "image/png", "audio/wav", "application/pdf"
	Data      string `json:"data"`
}

// Citation is a single citation entry on a text content block returned by the
// Messages API when a citation-enabled document was supplied.
type Citation struct {
	Type           string `json:"type"`
	CitedText      string `json:"cited_text"`
	DocumentIndex  int    `json:"document_index"`
	DocumentTitle  string `json:"document_title,omitempty"`
	StartCharIndex int    `json:"start_char_index,omitempty"`
	EndCharIndex   int    `json:"end_char_index,omitempty"`
}

type ToolDef struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	InputSchema  json.RawMessage `json:"input_schema"`
	CacheControl *CacheControl   `json:"cache_control,omitempty"`
}

// SSE payloads.

type sseMessageStart struct {
	Type    string         `json:"type"`
	Message sseMessageInfo `json:"message"`
}

type sseMessageInfo struct {
	ID    string   `json:"id"`
	Model string   `json:"model"`
	Usage sseUsage `json:"usage"`
}

type sseContentBlockStart struct {
	Type         string   `json:"type"`
	Index        int      `json:"index"`
	ContentBlock sseBlock `json:"content_block"`
}

type sseBlock struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
	Citations []Citation      `json:"citations,omitempty"`
}

type sseContentBlockDelta struct {
	Type  string   `json:"type"`
	Index int      `json:"index"`
	Delta sseDelta `json:"delta"`
}

type sseDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	Signature   string `json:"signature,omitempty"`
}

type sseContentBlockStop struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type sseMessageDelta struct {
	Type  string             `json:"type"`
	Delta sseMessageDeltaTop `json:"delta"`
	Usage sseUsage           `json:"usage"`
}

type sseMessageDeltaTop struct {
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
}

type sseUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

type sseError struct {
	Type  string         `json:"type"`
	Error sseErrorDetail `json:"error"`
}

type sseErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Conversion helpers.

func messagesFromCore(msgs []core.Message) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if len(m.Blocks) == 0 {
			continue
		}
		blocks := make([]Block, 0, len(m.Blocks))
		for _, b := range m.Blocks {
			switch v := b.(type) {
			case core.TextBlock:
				blocks = append(blocks, Block{Type: "text", Text: v.Text})
			case core.ToolUseBlock:
				input := v.Input
				if len(input) == 0 {
					input = json.RawMessage("{}")
				}
				blocks = append(blocks, Block{
					Type:  "tool_use",
					ID:    v.ID,
					Name:  v.Name,
					Input: input,
				})
			case core.ToolResultBlock:
				blocks = append(blocks, Block{
					Type:      "tool_result",
					ToolUseID: v.UseID,
					Content:   v.Content,
					IsError:   v.IsError,
				})
			case core.ThinkingBlock:
				blocks = append(blocks, Block{
					Type:      "thinking",
					Thinking:  v.Text,
					Signature: v.Signature,
				})
			case core.ImageBlock:
				blocks = append(blocks, Block{
					Type: "image",
					Source: &BlockSource{
						Type:      "base64",
						MediaType: v.MediaType,
						Data:      v.Source,
					},
				})
			case core.AudioBlock:
				blocks = append(blocks, Block{
					Type: "audio",
					Source: &BlockSource{
						Type:      "base64",
						MediaType: v.MediaType,
						Data:      v.Source,
					},
				})
			case core.DocumentBlock:
				blocks = append(blocks, Block{
					Type:  "document",
					Title: v.Title,
					Source: &BlockSource{
						Type:      "base64",
						MediaType: v.MediaType,
						Data:      v.Source,
					},
				})
			}
		}
		if len(blocks) == 0 {
			continue
		}
		out = append(out, Message{Role: string(m.Role), Content: blocks})
	}
	return out
}

// wireCitationsToCore converts wire-level citations into the core type used
// when attaching to TextBlock.
func wireCitationsToCore(in []Citation) []core.Citation {
	if len(in) == 0 {
		return nil
	}
	out := make([]core.Citation, 0, len(in))
	for _, c := range in {
		out = append(out, core.Citation{
			Type:          c.Type,
			CitedText:     c.CitedText,
			DocumentTitle: c.DocumentTitle,
			StartIndex:    c.StartCharIndex,
			EndIndex:      c.EndCharIndex,
		})
	}
	return out
}

func toolsFromCore(tools []core.Tool) []ToolDef {
	if len(tools) == 0 {
		return nil
	}
	out := make([]ToolDef, 0, len(tools))
	for _, t := range tools {
		out = append(out, ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.Schema(),
		})
	}
	out[len(out)-1].CacheControl = &CacheControl{Type: "ephemeral"}
	return out
}

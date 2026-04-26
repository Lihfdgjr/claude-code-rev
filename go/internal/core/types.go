package core

import "encoding/json"

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

type BlockKind string

const (
	KindText       BlockKind = "text"
	KindToolUse    BlockKind = "tool_use"
	KindToolResult BlockKind = "tool_result"
	KindThinking   BlockKind = "thinking"
	KindImage      BlockKind = "image"
	KindAudio      BlockKind = "audio"
	KindDocument   BlockKind = "document"
)

type Block interface {
	Kind() BlockKind
}

type TextBlock struct {
	Text      string
	Citations []Citation
}

func (TextBlock) Kind() BlockKind { return KindText }

// Citation describes one citation attached to a TextBlock by the model when
// the upstream request supplied a citation-enabled document.
type Citation struct {
	Type          string
	CitedText     string
	DocumentTitle string
	StartIndex    int
	EndIndex      int
}

type ToolUseBlock struct {
	ID    string
	Name  string
	Input json.RawMessage
}

func (ToolUseBlock) Kind() BlockKind { return KindToolUse }

// ImageBlock attaches an image to a user message.
// Source is base64-encoded; MediaType is e.g. "image/png".
type ImageBlock struct {
	Source    string
	MediaType string
}

func (ImageBlock) Kind() BlockKind { return KindImage }

// AudioBlock attaches audio. Source is base64; MediaType e.g. "audio/wav".
type AudioBlock struct {
	Source    string
	MediaType string
}

func (AudioBlock) Kind() BlockKind { return KindAudio }

// DocumentBlock attaches a PDF or text doc. Source is base64; MediaType e.g. "application/pdf".
type DocumentBlock struct {
	Source    string
	MediaType string
	Title     string
}

func (DocumentBlock) Kind() BlockKind { return KindDocument }

type ToolResultBlock struct {
	UseID   string
	Content string
	IsError bool
}

func (ToolResultBlock) Kind() BlockKind { return KindToolResult }

type ThinkingBlock struct {
	Text      string
	Signature string
}

func (ThinkingBlock) Kind() BlockKind { return KindThinking }

type Message struct {
	Role   Role
	Blocks []Block
}

type Usage struct {
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
}

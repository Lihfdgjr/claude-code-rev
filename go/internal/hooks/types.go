package hooks

import "encoding/json"

type EventName string

const (
	PreToolUse       EventName = "PreToolUse"
	PostToolUse      EventName = "PostToolUse"
	UserPromptSubmit EventName = "UserPromptSubmit"
	Stop             EventName = "Stop"
	SessionStart     EventName = "SessionStart"
	Notification     EventName = "Notification"
)

type HookSpec struct {
	Matcher string
	Type    string
	Command string
	Timeout int
}

type Config map[EventName][]HookSpec

type Event struct {
	Name       EventName
	ToolName   string
	ToolInput  json.RawMessage
	ToolOutput string
	UserText   string
	SessionID  string
}

type Decision struct {
	Block            bool
	Reason           string
	ReplacementInput json.RawMessage
}

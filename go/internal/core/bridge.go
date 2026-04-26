package core

// UIEvent is anything the chat layer pushes into the UI.
type UIEvent interface {
	UIEventKind() UIEventKind
}

type UIEventKind string

const (
	UIAssistantTextDelta UIEventKind = "assistant_text_delta"
	UIThinkingDelta      UIEventKind = "thinking_delta"
	UIToolStart          UIEventKind = "tool_start"
	UIToolResult         UIEventKind = "tool_result"
	UITurnDone           UIEventKind = "turn_done"
	UIStatus             UIEventKind = "status"
	UIError              UIEventKind = "error"
	UIPermissionPrompt   UIEventKind = "permission_prompt"
	UIAskUser            UIEventKind = "ask_user"
)

type UIAssistantTextDeltaEvent struct{ Text string }

func (UIAssistantTextDeltaEvent) UIEventKind() UIEventKind { return UIAssistantTextDelta }

type UIThinkingDeltaEvent struct{ Text string }

func (UIThinkingDeltaEvent) UIEventKind() UIEventKind { return UIThinkingDelta }

type UIToolStartEvent struct {
	ID    string
	Name  string
	Input string
}

func (UIToolStartEvent) UIEventKind() UIEventKind { return UIToolStart }

type UIToolResultEvent struct {
	ID      string
	Output  string
	IsError bool
}

func (UIToolResultEvent) UIEventKind() UIEventKind { return UIToolResult }

type UITurnDoneEvent struct {
	StopReason string
	Usage      Usage
}

func (UITurnDoneEvent) UIEventKind() UIEventKind { return UITurnDone }

type UIStatusEvent struct {
	Level NotifyLevel
	Text  string
}

func (UIStatusEvent) UIEventKind() UIEventKind { return UIStatus }

type UIErrorEvent struct{ Err error }

func (UIErrorEvent) UIEventKind() UIEventKind { return UIError }

type UIPermissionPromptEvent struct {
	Tool       string
	InputJSON  string
	ReplyChan  chan PermissionResponse
	RememberOK bool
}

func (UIPermissionPromptEvent) UIEventKind() UIEventKind { return UIPermissionPrompt }

// UIAskUserEvent is pushed by an interactive tool that needs user input.
// The tool blocks on Reply (string) until the UI responds. Empty string ⇒ cancelled.
type UIAskUserEvent struct {
	Question string
	Reply    chan string
}

func (UIAskUserEvent) UIEventKind() UIEventKind { return UIAskUser }

// Driver is what the UI calls to interact with the chat layer.
type Driver interface {
	Submit(text string) <-chan UIEvent
	Cancel()
	Snapshot() []Message
	RunCommand(line string) error
	Notifications() <-chan UIEvent
	Session() Session
	Commands() CommandRegistry
	Tools() ToolRegistry
}

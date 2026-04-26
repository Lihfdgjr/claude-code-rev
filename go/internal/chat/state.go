package chat

import "sync/atomic"

// ThinkingEnabled toggles extended thinking on outgoing requests.
var ThinkingEnabled atomic.Bool

// PlanModeActive blocks mutating tools (Write/Edit/Bash...) when true.
var PlanModeActive atomic.Bool

// PlanWriteTools is the set of tools blocked while plan mode is active.
var PlanWriteTools = map[string]bool{
	"Write":         true,
	"Edit":          true,
	"MultiEdit":     true,
	"NotebookEdit":  true,
	"Bash":          true,
	"BashOutput":    true,
	"KillBash":      true,
	"FilesUpload":   true,
	"BatchSubmit":   true,
	"ComputerUse":   true,
}

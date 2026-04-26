package commands

import (
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/hooks"
	"claudecode/internal/memory"
	"claudecode/internal/oauth"
	"claudecode/internal/sessions"
)

type Registry struct {
	cmds  []core.Command
	index map[string]core.Command
}

// Deps bundles the dependencies needed by stateful commands.
type Deps struct {
	SessionStore   *sessions.Store
	OAuthStore     *oauth.Store
	HooksCfg       hooks.Config
	MemoryStore    *memory.Store
	Transport      core.Transport
	TranscriptRoot string
}

func New(cmds []core.Command) *Registry {
	r := &Registry{index: make(map[string]core.Command, len(cmds))}
	for _, c := range cmds {
		r.Add(c)
	}
	return r
}

func (r *Registry) Add(cmd core.Command) {
	if cmd == nil {
		return
	}
	name := cmd.Name()
	if _, exists := r.index[name]; exists {
		return
	}
	r.cmds = append(r.cmds, cmd)
	r.index[name] = cmd
}

func (r *Registry) Get(name string) (core.Command, bool) {
	c, ok := r.index[name]
	return c, ok
}

func (r *Registry) All() []core.Command {
	out := make([]core.Command, len(r.cmds))
	copy(out, r.cmds)
	return out
}

func (r *Registry) Parse(line string) (cmd core.Command, args string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, "", false
	}
	name, rest, _ := strings.Cut(line, " ")
	c, found := r.index[name]
	if !found {
		return nil, "", false
	}
	return c, strings.TrimSpace(rest), true
}

// Default builds the full command registry with all built-ins.
func Default(deps Deps) *Registry {
	cmds := []core.Command{
		// Session control
		NewClear(),
		NewReset(),
		NewCompact(),
		NewModel(),

		// Project
		NewInit(),
		NewMemory(),

		// Status / inspection
		NewVersion(),
		NewStatus(),
		NewDoctor(),
		NewConfig(),
		NewCost(),
		NewEnv(),
		NewReleaseNotes(),

		// Discovery
		NewAgents(),
		NewSkills(),
		NewMCP(),

		// Reviews
		NewReview(),
		NewSecurityReview(),
		NewPRComments(),

		// Settings
		NewPermissions(),
		NewPrivacySettings(),
		NewVim(),
		NewFast(),
		NewThinking(),
		NewIDE(),
		NewComputerUseCmd(),

		// Feedback
		NewBug(),
		NewFeedback(),

		// History / persistence
		NewTranscript(),
		NewExport(),
		NewDump(),
		NewMessages(),
		NewImport(),
		NewSummary(),

		// Git helpers
		NewCommit(),
		NewCreatePR(),
		NewDiff(),
		NewGitStatus(),
		NewPush(),

		// Misc info / utilities
		NewAddDir(),
		NewAllowedTools(),
		NewAuth(),
		NewBashes(),
		NewEditor(),
		NewLogs(),
		NewNew(),
		NewSync(),
		NewSystem(),
		NewTasks(),
		NewTheme(),
		NewSettings(),
		NewKeybindings(),
		NewTimestamp(),
		NewTools(),
		NewUsage(),
		NewWatch(),
		NewJSON(),

		// Round 9: branching, retry, attachments
		NewBranch(),
		NewRetry(),
		NewCancel(),
		NewTitle(),
		NewImage(),
		NewAudio(),
		NewDocument(),
		NewClipboard(),
		NewAttachments(),
		NewClearAttachments(),

		// Round 13b: autoCompact / transcript / hot-reload / shell / workspace / undo
		NewTokens(),
		NewReload(),
		NewWorkspace(),
		NewShell(),
		NewUndo(),
		NewRedo(),
		NewCheckpoint(),
	}

	if deps.SessionStore != nil {
		cmds = append(cmds, NewResume(deps.SessionStore), NewHistory(deps.SessionStore), NewSave(deps.SessionStore), NewFind(deps.SessionStore))
		cmds = append(cmds, NewRecover(deps.SessionStore, deps.TranscriptRoot))
	}
	if deps.OAuthStore != nil {
		cmds = append(cmds, NewLogin(deps.OAuthStore), NewLogout(deps.OAuthStore))
	}
	cmds = append(cmds, NewHooks(deps.HooksCfg))
	if deps.MemoryStore != nil {
		cmds = append(cmds, NewDream(deps.MemoryStore, deps.Transport))
	}

	reg := New(cmds)
	reg.Add(NewHelp(reg))
	return reg
}

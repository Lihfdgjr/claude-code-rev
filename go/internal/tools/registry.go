package tools

import "claudecode/internal/core"

type Registry struct {
	tools  []core.Tool
	byName map[string]core.Tool
}

func New(tools []core.Tool) *Registry {
	r := &Registry{
		tools:  make([]core.Tool, 0, len(tools)),
		byName: make(map[string]core.Tool, len(tools)),
	}
	for _, t := range tools {
		if t == nil {
			continue
		}
		if _, exists := r.byName[t.Name()]; exists {
			continue
		}
		r.tools = append(r.tools, t)
		r.byName[t.Name()] = t
	}
	return r
}

// Default returns the built-in stateless tool set. Tools that need
// dependencies (Agent, Skill, MCP) must be appended by the caller.
func Default() *Registry {
	return New([]core.Tool{
		NewRead(),
		NewWrite(),
		NewEdit(),
		NewMultiEdit(),
		NewLS(),
		NewBash(),
		NewBashOutput(),
		NewKillBash(),
		NewGrep(),
		NewGlob(),
		NewTodoWrite(),
		NewNotebookRead(),
		NewNotebookEdit(),
		NewWebFetch(),
		NewWebSearch(),
		NewFilesUpload(),
		NewBatchSubmit(),
		NewComputerUse(nil),
		NewLSPDefinition(),
		NewLSPHover(),
		NewLSPReferences(),
		NewLSPSymbols(),
		NewEnterPlanMode(),
		NewExitPlanMode(),
		NewAskUser(),
		NewTaskCreate(),
		NewTaskUpdate(),
		NewTaskList(),
		NewScheduleWakeup(),
		NewWorktreeCreate(),
		NewWorktreeRemove(),
		NewReadManyFiles(),
		NewHTTPRequest(),
		NewGitDiff(),
		NewGitLog(),
		NewGitBlame(),
		NewGitCommit(),
		NewPatch(),
		NewTokenCount(),
		NewFileWatch(),
		NewCalculator(),
		NewDNSLookup(),
		NewTextDiff(),
	})
}

func (r *Registry) Get(name string) (core.Tool, bool) {
	t, ok := r.byName[name]
	return t, ok
}

func (r *Registry) All() []core.Tool {
	out := make([]core.Tool, len(r.tools))
	copy(out, r.tools)
	return out
}

// Add appends an extra tool to the registry (e.g., Agent, Skill, MCP wrappers).
// Duplicates by name are ignored.
func (r *Registry) Add(t core.Tool) {
	if t == nil {
		return
	}
	if _, exists := r.byName[t.Name()]; exists {
		return
	}
	r.tools = append(r.tools, t)
	r.byName[t.Name()] = t
}

package chat

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"claudecode/internal/memory"
)

func BuildSystemPrompt(mem *memory.Memory) string {
	var b strings.Builder
	b.WriteString("You are claudecode-go, a coding assistant operating in a terminal CLI.\n")
	b.WriteString("Use the provided tools to read, search, write, and edit files, and to run shell commands.\n")
	b.WriteString("Make minimal, surgical changes. Confirm destructive actions before running them.\n")
	b.WriteString("Communicate concisely; output is rendered in a TUI.\n\n")

	cwd, _ := os.Getwd()
	fmt.Fprintf(&b, "Environment:\n- Working directory: %s\n- OS: %s/%s\n\n", cwd, runtime.GOOS, runtime.GOARCH)

	if mem != nil {
		if c := mem.Combined(); c != "" {
			b.WriteString(c)
			b.WriteString("\n")
		}
	}
	return b.String()
}

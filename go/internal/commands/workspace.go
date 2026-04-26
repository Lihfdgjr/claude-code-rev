package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"claudecode/internal/core"
	"claudecode/internal/workspace"
)

type workspaceCmd struct{}

func NewWorkspace() core.Command { return &workspaceCmd{} }

func (workspaceCmd) Name() string     { return "workspace" }
func (workspaceCmd) Synopsis() string { return "Show detected workspace metadata" }

func (workspaceCmd) Run(ctx context.Context, args string, sess core.Session) error {
	cwd, _ := os.Getwd()
	w := workspace.Detect(cwd)

	var b strings.Builder
	fmt.Fprintf(&b, "root: %s\n", w.Root)
	fmt.Fprintf(&b, "kind: %s\n", w.Kind)
	if len(w.Languages) > 0 {
		fmt.Fprintf(&b, "languages: %s\n", strings.Join(w.Languages, ", "))
	} else {
		b.WriteString("languages: (none detected)\n")
	}
	manifests := []string{}
	if w.HasGoMod {
		manifests = append(manifests, "go.mod")
	}
	if w.HasPackageJSON {
		manifests = append(manifests, "package.json")
	}
	if w.HasCargoToml {
		manifests = append(manifests, "Cargo.toml")
	}
	if w.HasPyProject {
		manifests = append(manifests, "pyproject.toml")
	}
	if len(manifests) > 0 {
		fmt.Fprintf(&b, "manifests: %s", strings.Join(manifests, ", "))
	} else {
		b.WriteString("manifests: (none)")
	}
	sess.Notify(core.NotifyInfo, b.String())
	return nil
}

package commands

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"claudecode/internal/core"
)

type envCmd struct{}

func NewEnv() core.Command { return &envCmd{} }

func (envCmd) Name() string     { return "env" }
func (envCmd) Synopsis() string { return "Show relevant environment variables" }

func (envCmd) Run(ctx context.Context, args string, sess core.Session) error {
	prefixes := []string{"ANTHROPIC_", "CLAUDECODE_"}
	extras := map[string]bool{"NO_COLOR": true, "TERM": true}

	out := map[string]string{}
	for _, kv := range os.Environ() {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			continue
		}
		k, v := kv[:i], kv[i+1:]
		match := extras[k]
		if !match {
			for _, p := range prefixes {
				if strings.HasPrefix(k, p) {
					match = true
					break
				}
			}
		}
		if match {
			out[k] = v
		}
	}

	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		v := out[k]
		if strings.Contains(strings.ToLower(k), "key") || strings.Contains(strings.ToLower(k), "token") {
			v = maskSecret(v)
		}
		fmt.Fprintf(&b, "%s=%s\n", k, v)
	}
	if b.Len() == 0 {
		sess.Notify(core.NotifyInfo, "(no relevant env vars set)")
		return nil
	}
	sess.Notify(core.NotifyInfo, strings.TrimRight(b.String(), "\n"))
	return nil
}

func maskSecret(s string) string {
	if len(s) <= 10 {
		return strings.Repeat("*", len(s))
	}
	return s[:6] + strings.Repeat("*", len(s)-10) + s[len(s)-4:]
}

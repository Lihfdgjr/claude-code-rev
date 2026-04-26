package commands

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"claudecode/internal/core"
)

type doctorCmd struct{}

func NewDoctor() core.Command { return &doctorCmd{} }

func (doctorCmd) Name() string     { return "doctor" }
func (doctorCmd) Synopsis() string { return "Run environment health checks" }

func (doctorCmd) Run(ctx context.Context, args string, sess core.Session) error {
	var b strings.Builder

	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		b.WriteString("PASS  ANTHROPIC_API_KEY set\n")
	} else {
		b.WriteString("FAIL  ANTHROPIC_API_KEY missing\n")
	}

	netCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var resolver net.Resolver
	if _, err := resolver.LookupHost(netCtx, "api.anthropic.com"); err == nil {
		b.WriteString("PASS  DNS api.anthropic.com\n")
	} else {
		fmt.Fprintf(&b, "FAIL  DNS api.anthropic.com: %v\n", err)
	}

	fmt.Fprintf(&b, "PASS  Go runtime %s", runtime.Version())

	sess.Notify(core.NotifyInfo, b.String())
	return nil
}

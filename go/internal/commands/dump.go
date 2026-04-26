package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"claudecode/internal/core"
)

type dumpCmd struct{}

func NewDump() core.Command { return &dumpCmd{} }

func (dumpCmd) Name() string     { return "dump" }
func (dumpCmd) Synopsis() string { return "Dump conversation history as JSON" }

func (dumpCmd) Run(ctx context.Context, args string, sess core.Session) error {
	hist := sess.History()
	type blockOut struct {
		Kind string      `json:"kind"`
		Data interface{} `json:"data"`
	}
	type msgOut struct {
		Role   core.Role  `json:"role"`
		Blocks []blockOut `json:"blocks"`
	}
	out := make([]msgOut, 0, len(hist))
	for _, m := range hist {
		blocks := make([]blockOut, 0, len(m.Blocks))
		for _, b := range m.Blocks {
			blocks = append(blocks, blockOut{Kind: string(b.Kind()), Data: b})
		}
		out = append(out, msgOut{Role: m.Role, Blocks: blocks})
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("marshal: %v", err))
		return err
	}
	s := string(data)
	if len(s) > 5000 {
		s = s[:5000] + "\n... (truncated)"
	}
	sess.Notify(core.NotifyInfo, s)
	return nil
}

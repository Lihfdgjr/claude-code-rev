package commands

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claudecode/internal/core"
)

const maxDocumentBytes = 32 * 1024 * 1024

type documentCmd struct{}

func NewDocument() core.Command { return &documentCmd{} }

func (documentCmd) Name() string     { return "document" }
func (documentCmd) Synopsis() string { return "Attach a document (pdf/txt) to the next message" }

func (documentCmd) Run(ctx context.Context, args string, sess core.Session) error {
	path := strings.TrimSpace(args)
	if path == "" {
		sess.Notify(core.NotifyWarn, "Usage: /document <path>")
		return nil
	}
	mt, ok := documentMediaType(path)
	if !ok {
		sess.Notify(core.NotifyError, "unsupported document type (use .pdf, .txt)")
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("stat %s: %v", path, err))
		return nil
	}
	if info.Size() > maxDocumentBytes {
		sess.Notify(core.NotifyError, fmt.Sprintf("document too large: %d bytes (max %d)", info.Size(), maxDocumentBytes))
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("read %s: %v", path, err))
		return nil
	}
	enc := base64.StdEncoding.EncodeToString(data)
	sess.Attach(core.DocumentBlock{
		Source:    enc,
		MediaType: mt,
		Title:     filepath.Base(path),
	})
	sess.Notify(core.NotifyInfo, fmt.Sprintf("attached document %q (%d bytes); will be sent with your next message.", filepath.Base(path), len(data)))
	return nil
}

func documentMediaType(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return "application/pdf", true
	case ".txt":
		return "text/plain", true
	}
	return "", false
}

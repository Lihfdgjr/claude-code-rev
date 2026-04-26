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

const maxImageBytes = 5 * 1024 * 1024

type imageCmd struct{}

func NewImage() core.Command { return &imageCmd{} }

func (imageCmd) Name() string     { return "image" }
func (imageCmd) Synopsis() string { return "Attach an image (png/jpg/gif/webp) to the next message" }

func (imageCmd) Run(ctx context.Context, args string, sess core.Session) error {
	path := strings.TrimSpace(args)
	if path == "" {
		sess.Notify(core.NotifyWarn, "Usage: /image <path>")
		return nil
	}
	mt, ok := imageMediaType(path)
	if !ok {
		sess.Notify(core.NotifyError, "unsupported image type (use .png, .jpg, .jpeg, .gif, .webp)")
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("stat %s: %v", path, err))
		return nil
	}
	if info.Size() > maxImageBytes {
		sess.Notify(core.NotifyError, fmt.Sprintf("image too large: %d bytes (max %d)", info.Size(), maxImageBytes))
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("read %s: %v", path, err))
		return nil
	}
	enc := base64.StdEncoding.EncodeToString(data)
	sess.Attach(core.ImageBlock{Source: enc, MediaType: mt})
	sess.Notify(core.NotifyInfo, fmt.Sprintf("attached image (%d bytes); will be sent with your next message.", len(data)))
	return nil
}

func imageMediaType(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png", true
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".gif":
		return "image/gif", true
	case ".webp":
		return "image/webp", true
	}
	return "", false
}

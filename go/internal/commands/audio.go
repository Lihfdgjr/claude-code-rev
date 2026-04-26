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

const maxAudioBytes = 25 * 1024 * 1024

type audioCmd struct{}

func NewAudio() core.Command { return &audioCmd{} }

func (audioCmd) Name() string     { return "audio" }
func (audioCmd) Synopsis() string { return "Attach an audio file (wav/mp3/ogg) to the next message" }

func (audioCmd) Run(ctx context.Context, args string, sess core.Session) error {
	path := strings.TrimSpace(args)
	if path == "" {
		sess.Notify(core.NotifyWarn, "Usage: /audio <path>")
		return nil
	}
	mt, ok := audioMediaType(path)
	if !ok {
		sess.Notify(core.NotifyError, "unsupported audio type (use .wav, .mp3, .ogg)")
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("stat %s: %v", path, err))
		return nil
	}
	if info.Size() > maxAudioBytes {
		sess.Notify(core.NotifyError, fmt.Sprintf("audio too large: %d bytes (max %d)", info.Size(), maxAudioBytes))
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		sess.Notify(core.NotifyError, fmt.Sprintf("read %s: %v", path, err))
		return nil
	}
	enc := base64.StdEncoding.EncodeToString(data)
	sess.Attach(core.AudioBlock{Source: enc, MediaType: mt})
	sess.Notify(core.NotifyInfo, fmt.Sprintf("attached audio (%d bytes); will be sent with your next message.", len(data)))
	return nil
}

func audioMediaType(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".wav":
		return "audio/wav", true
	case ".mp3":
		return "audio/mpeg", true
	case ".ogg":
		return "audio/ogg", true
	}
	return "", false
}

package audio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// ErrNotSupported is returned when audio capture is not available.
var ErrNotSupported = errors.New("audio: not supported in this build")

// Recorder captures microphone input.
type Recorder interface {
	Start(ctx context.Context) error
	Stop() ([]byte, error)
	IsRecording() bool
}

type ffmpegRecorder struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	tmpPath   string
	recording bool
	done      chan error
}

// New returns a Recorder backed by ffmpeg.
func New() Recorder { return &ffmpegRecorder{} }

func (r *ffmpegRecorder) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.recording {
		return errors.New("audio: already recording")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return errors.New("audio: ffmpeg not found in PATH")
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("ccaudio_%d.wav", time.Now().UnixNano()))
	var args []string
	switch runtime.GOOS {
	case "windows":
		args = []string{"-hide_banner", "-loglevel", "error", "-f", "dshow", "-i", `audio=default`, "-ac", "1", "-ar", "16000", "-y", tmp}
	case "linux":
		args = []string{"-hide_banner", "-loglevel", "error", "-f", "alsa", "-i", "default", "-ac", "1", "-ar", "16000", "-y", tmp}
	case "darwin":
		args = []string{"-hide_banner", "-loglevel", "error", "-f", "avfoundation", "-i", ":0", "-ac", "1", "-ar", "16000", "-y", tmp}
	default:
		return ErrNotSupported
	}
	cmd := exec.Command("ffmpeg", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("audio: stdin pipe: %w", err)
	}
	_ = stdin
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("audio: start ffmpeg: %w", err)
	}
	r.cmd = cmd
	r.tmpPath = tmp
	r.recording = true
	r.done = make(chan error, 1)
	go func() { r.done <- cmd.Wait() }()
	return nil
}

func (r *ffmpegRecorder) Stop() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.recording {
		return nil, errors.New("audio: not recording")
	}
	defer func() {
		r.recording = false
		r.cmd = nil
		if r.tmpPath != "" {
			os.Remove(r.tmpPath)
		}
		r.tmpPath = ""
	}()

	// Try a graceful interrupt; on Windows os.Interrupt typically cannot be
	// delivered to a child, so fall back to Kill if the process is still
	// running after the grace window.
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Signal(os.Interrupt)
		select {
		case <-r.done:
		case <-time.After(3 * time.Second):
			_ = r.cmd.Process.Kill()
			<-r.done
		}
	}

	data, err := os.ReadFile(r.tmpPath)
	if err != nil {
		return nil, fmt.Errorf("audio: read recording: %w", err)
	}
	return data, nil
}

func (r *ffmpegRecorder) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recording
}

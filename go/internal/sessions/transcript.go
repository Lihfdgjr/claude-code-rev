package sessions

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Recorder appends JSONL transcript events for a session.
type Recorder struct {
	mu   sync.Mutex
	f    *os.File
	path string
}

// transcriptEntry is one JSONL line.
type transcriptEntry struct {
	TS   time.Time   `json:"ts"`
	Kind string      `json:"kind"`
	Data interface{} `json:"data"`
}

// RecorderPath returns the canonical transcript path for a session.
func RecorderPath(rootDir, sessionID string) string {
	return filepath.Join(rootDir, "transcripts", sessionID+".jsonl")
}

// NewRecorder opens (or creates) the transcript file for sessionID.
func NewRecorder(rootDir, sessionID string) (*Recorder, error) {
	if sessionID == "" {
		return nil, errors.New("sessions: empty session id")
	}
	path := RecorderPath(rootDir, sessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Recorder{f: f, path: path}, nil
}

// Write appends one JSONL line and syncs.
func (r *Recorder) Write(eventKind string, data interface{}) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return errors.New("sessions: recorder closed")
	}
	entry := transcriptEntry{
		TS:   time.Now().UTC(),
		Kind: eventKind,
		Data: data,
	}
	buf, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	if _, err := r.f.Write(buf); err != nil {
		return err
	}
	return r.f.Sync()
}

// Close releases the underlying file handle.
func (r *Recorder) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return nil
	}
	err := r.f.Close()
	r.f = nil
	return err
}

// Path returns the on-disk transcript path.
func (r *Recorder) Path() string {
	if r == nil {
		return ""
	}
	return r.path
}

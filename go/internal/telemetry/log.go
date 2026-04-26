package telemetry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Event struct {
	Time time.Time              `json:"time"`
	Kind string                 `json:"kind"`
	Data map[string]interface{} `json:"data,omitempty"`
}

type Logger struct {
	path string
	mu   sync.Mutex
	f    *os.File
}

// New opens (or creates) an append-only JSONL file at path. The parent
// directory is created if missing. The file is opened lazily on the first
// Log call so construction itself never fails.
func New(path string) *Logger {
	return &Logger{path: path}
}

// DefaultPath returns ~/.claude/telemetry.jsonl, falling back to the
// current working directory if the user home cannot be resolved.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "telemetry.jsonl"
	}
	return filepath.Join(home, ".claude", "telemetry.jsonl")
}

func (l *Logger) open() error {
	if l.f != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	l.f = f
	return nil
}

func (l *Logger) Log(e Event) error {
	if l == nil {
		return errors.New("telemetry: nil logger")
	}
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.open(); err != nil {
		return err
	}
	if _, err := l.f.Write(data); err != nil {
		return err
	}
	return l.f.Sync()
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

// Global logger plumbing — packages that don't want to thread a Logger
// through can call SetGlobal once at startup and Global() from anywhere.
var (
	globalMu sync.RWMutex
	global   *Logger
)

func SetGlobal(l *Logger) {
	globalMu.Lock()
	global = l
	globalMu.Unlock()
}

func Global() *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// LogGlobal is a nil-safe convenience wrapper around Global().Log. It silently
// drops events when no global logger has been installed so callers can sprinkle
// telemetry calls without guarding each site.
func LogGlobal(kind string, data map[string]interface{}) {
	l := Global()
	if l == nil {
		return
	}
	_ = l.Log(Event{Kind: kind, Data: data})
}

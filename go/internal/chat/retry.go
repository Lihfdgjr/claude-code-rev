package chat

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"claudecode/internal/core"
)

const (
	defaultMaxAttempts = 4
	baseBackoff        = 250 * time.Millisecond
	maxBackoff         = 8 * time.Second
)

var transientMarkers = []string{
	"429",
	"500",
	"502",
	"503",
	"504",
	"connection reset",
	"EOF",
	"timeout",
}

// StreamWithRetry wraps transport.Stream with exponential backoff for transient
// errors (HTTP 429, 5xx, network reset, EOF, timeout).
func StreamWithRetry(ctx context.Context, transport core.Transport, opts core.CallOptions, history []core.Message, maxAttempts int) (<-chan core.StreamEvent, error) {
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ch, err := transport.Stream(ctx, opts, history)
		if err == nil {
			return ch, nil
		}
		lastErr = err
		if !isTransient(err) {
			return nil, err
		}
		if attempt == maxAttempts-1 {
			break
		}
		if err := sleep(ctx, backoffFor(attempt)); err != nil {
			return nil, err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("stream: retry exhausted")
	}
	return nil, lastErr
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, m := range transientMarkers {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
}

func backoffFor(attempt int) time.Duration {
	d := baseBackoff << attempt
	if d <= 0 || d > maxBackoff {
		d = maxBackoff
	}
	jitter := time.Duration(rand.Int63n(int64(100*time.Millisecond))) - 50*time.Millisecond
	d += jitter
	if d < 0 {
		d = 0
	}
	return d
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"claudecode/internal/core"
)

// AutoCompactConfig controls when AutoCompact decides to summarize.
type AutoCompactConfig struct {
	Threshold float64
	MinPeriod time.Duration
}

// AutoCompact watches session history and triggers Compact when the budget
// percentage crosses the threshold, while rate-limiting back-to-back runs.
type AutoCompact struct {
	cfg        AutoCompactConfig
	mu         sync.Mutex
	lastRun    time.Time
	inFlight   bool
	NotifyHook func(success bool, msg string)
}

// NewAutoCompact builds an AutoCompact with sensible defaults if cfg is zero.
func NewAutoCompact(cfg AutoCompactConfig) *AutoCompact {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 0.75
	}
	if cfg.MinPeriod <= 0 {
		cfg.MinPeriod = 2 * time.Minute
	}
	return &AutoCompact{cfg: cfg}
}

// MaybeRun returns true if a compact run was scheduled. The actual work
// happens on a background goroutine.
func (a *AutoCompact) MaybeRun(ctx context.Context, sess core.Session) bool {
	if a == nil || sess == nil {
		return false
	}

	pct := BudgetPercent(sess.History(), sess.Model())
	if pct < a.cfg.Threshold {
		return false
	}

	a.mu.Lock()
	if a.inFlight {
		a.mu.Unlock()
		return false
	}
	if !a.lastRun.IsZero() && time.Since(a.lastRun) < a.cfg.MinPeriod {
		a.mu.Unlock()
		return false
	}
	a.inFlight = true
	a.mu.Unlock()

	go func() {
		err := sess.Compact(ctx)

		a.mu.Lock()
		a.inFlight = false
		a.lastRun = time.Now()
		hook := a.NotifyHook
		a.mu.Unlock()

		if hook != nil {
			if err != nil {
				hook(false, fmt.Sprintf("autoCompact failed: %v", err))
			} else {
				hook(true, "autoCompact: conversation summarized")
			}
		}
	}()
	return true
}

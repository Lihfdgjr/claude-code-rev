package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"claudecode/internal/core"
)

// schedule_wakeup uses two notification channels: per-job UI events (best-effort,
// captured ctx may be invalid past the originating turn) AND a package-level
// pending-notifications buffer that the host can drain on next /status. The
// recover() guards the UI-channel send against closed/nil chans.

type wakeupJob struct {
	deadline time.Time
	reason   string
	prompt   string
	ui       chan<- core.UIEvent
}

var (
	wakeupOnce    sync.Once
	wakeupMu      sync.Mutex
	wakeupJobs    []wakeupJob
	wakeupPending []string
)

func startWakeupScheduler() {
	wakeupOnce.Do(func() {
		go wakeupLoop()
	})
}

func wakeupLoop() {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for now := range tick.C {
		wakeupMu.Lock()
		var remaining []wakeupJob
		var due []wakeupJob
		for _, j := range wakeupJobs {
			if !now.Before(j.deadline) {
				due = append(due, j)
			} else {
				remaining = append(remaining, j)
			}
		}
		wakeupJobs = remaining
		for _, j := range due {
			msg := "wakeup: " + j.reason
			wakeupPending = append(wakeupPending, msg)
		}
		wakeupMu.Unlock()
		for _, j := range due {
			deliverWakeup(j)
		}
	}
}

func deliverWakeup(j wakeupJob) {
	defer func() { _ = recover() }()
	if j.ui == nil {
		return
	}
	ev := core.UIStatusEvent{Level: core.NotifyInfo, Text: "wakeup: " + j.reason}
	select {
	case j.ui <- ev:
	default:
	}
}

// DrainPending returns and clears the buffered wakeup notifications. Hosts can
// call this from /status or the next-turn boundary to surface scheduled wakeups
// even when the originating turn's UI channel has gone away.
func DrainPending() []string {
	wakeupMu.Lock()
	defer wakeupMu.Unlock()
	out := wakeupPending
	wakeupPending = nil
	return out
}

type scheduleWakeupTool struct{}

func NewScheduleWakeup() core.Tool { return &scheduleWakeupTool{} }

func (t *scheduleWakeupTool) Name() string { return "ScheduleWakeup" }

func (t *scheduleWakeupTool) Description() string {
	return "Schedule a wakeup notification N seconds in the future (60..3600). Reason is required; an optional prompt is recorded. Notifications fire as UIStatusEvent and are also buffered for DrainPending()."
}

func (t *scheduleWakeupTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"delaySeconds": {"type": "integer", "minimum": 60, "maximum": 3600},
			"reason": {"type": "string"},
			"prompt": {"type": "string"}
		},
		"required": ["delaySeconds", "reason"],
		"additionalProperties": false
	}`)
}

func (t *scheduleWakeupTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		DelaySeconds int    `json:"delaySeconds"`
		Reason       string `json:"reason"`
		Prompt       string `json:"prompt"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if args.DelaySeconds < 60 || args.DelaySeconds > 3600 {
		return "", fmt.Errorf("delaySeconds must be between 60 and 3600 (got %d)", args.DelaySeconds)
	}
	if args.Reason == "" {
		return "", fmt.Errorf("reason required")
	}

	startWakeupScheduler()
	deadline := time.Now().Add(time.Duration(args.DelaySeconds) * time.Second)
	job := wakeupJob{
		deadline: deadline,
		reason:   args.Reason,
		prompt:   args.Prompt,
		ui:       core.UIEvents(ctx),
	}
	wakeupMu.Lock()
	wakeupJobs = append(wakeupJobs, job)
	wakeupMu.Unlock()

	return fmt.Sprintf("scheduled wakeup at %s: %s", deadline.Format(time.RFC3339), args.Reason), nil
}

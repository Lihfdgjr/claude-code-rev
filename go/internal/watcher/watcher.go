package watcher

import (
	"os"
	"sync"
	"time"
)

type target struct {
	Path      string
	Mtime     time.Time
	Size      int64
	Callbacks []func(path string)
}

type Watcher struct {
	mu      sync.Mutex
	targets map[string]*target
	period  time.Duration
	stop    chan struct{}
	running bool
}

func New(period time.Duration) *Watcher {
	if period <= 0 {
		period = 2 * time.Second
	}
	return &Watcher{
		targets: make(map[string]*target),
		period:  period,
		stop:    make(chan struct{}),
	}
}

func (w *Watcher) Add(path string, cb func(path string)) {
	if path == "" || cb == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	t, ok := w.targets[path]
	if !ok {
		t = &target{Path: path}
		if info, err := os.Stat(path); err == nil {
			t.Mtime = info.ModTime()
			t.Size = info.Size()
		}
		w.targets[path] = t
	}
	t.Callbacks = append(t.Callbacks, cb)
}

func (w *Watcher) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()
	go w.loop()
}

func (w *Watcher) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()
	close(w.stop)
}

func (w *Watcher) loop() {
	ticker := time.NewTicker(w.period)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			w.tick()
		}
	}
}

func (w *Watcher) tick() {
	w.mu.Lock()
	type fire struct {
		path string
		cbs  []func(path string)
	}
	var fires []fire
	for _, t := range w.targets {
		var mtime time.Time
		var size int64
		info, err := os.Stat(t.Path)
		if err == nil {
			mtime = info.ModTime()
			size = info.Size()
		}
		if !mtime.Equal(t.Mtime) || size != t.Size {
			t.Mtime = mtime
			t.Size = size
			cbs := make([]func(path string), len(t.Callbacks))
			copy(cbs, t.Callbacks)
			fires = append(fires, fire{path: t.Path, cbs: cbs})
		}
	}
	w.mu.Unlock()

	for _, f := range fires {
		for _, cb := range f.cbs {
			cb(f.path)
		}
	}
}

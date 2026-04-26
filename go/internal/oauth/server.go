package oauth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

const callbackTimeout = 5 * time.Minute

// ListenForCode starts a localhost HTTP server on 127.0.0.1:port and waits
// for an OAuth provider to redirect to /callback with ?code=&state=. It
// returns the captured values, then shuts the server down cleanly.
func ListenForCode(ctx context.Context, port int) (string, string, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", "", fmt.Errorf("oauth: listen %s: %w", addr, err)
	}

	type result struct {
		code  string
		state string
		err   error
	}
	ch := make(chan result, 1)
	var once sync.Once
	deliver := func(r result) { once.Do(func() { ch <- r }) }

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if e := q.Get("error"); e != "" {
			http.Error(w, "OAuth error: "+e, http.StatusBadRequest)
			deliver(result{err: fmt.Errorf("oauth: provider error: %s", e)})
			return
		}
		code := q.Get("code")
		state := q.Get("state")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			deliver(result{err: errors.New("oauth: missing code in callback")})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!doctype html><html><body><h2>Authentication complete</h2><p>You may close this window.</p></body></html>"))
		deliver(result{code: code, state: state})
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			deliver(result{err: err})
		}
	}()

	shutdown := func() {
		sctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}

	select {
	case r := <-ch:
		shutdown()
		return r.code, r.state, r.err
	case <-ctx.Done():
		shutdown()
		return "", "", ctx.Err()
	case <-time.After(callbackTimeout):
		shutdown()
		return "", "", errors.New("oauth: timed out waiting for callback")
	}
}

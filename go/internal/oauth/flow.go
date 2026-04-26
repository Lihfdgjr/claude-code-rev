package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Flow struct {
	AuthURL  string
	TokenURL string
	ClientID string
	Port     int // default 53682
}

func NewFlow(authURL, tokenURL, clientID string) *Flow {
	return &Flow{AuthURL: authURL, TokenURL: tokenURL, ClientID: clientID, Port: 53682}
}

// Begin opens the browser at the auth URL, then blocks waiting for the
// callback. Returns the auth code (or error).
func (f *Flow) Begin(ctx context.Context) (string, error) {
	if f.AuthURL == "" {
		return "", errors.New("oauth: AuthURL not configured. Use /login <token> to set a token directly, or set ANTHROPIC_API_KEY env var.")
	}
	state := randomState()
	redirect := fmt.Sprintf("http://127.0.0.1:%d/callback", f.Port)
	u := f.AuthURL + "?" + url.Values{
		"client_id":     []string{f.ClientID},
		"redirect_uri":  []string{redirect},
		"response_type": []string{"code"},
		"state":         []string{state},
	}.Encode()
	_ = OpenBrowser(u) // user can copy-paste if launcher fails
	rcvCode, rcvState, err := ListenForCode(ctx, f.Port)
	if err != nil {
		return "", err
	}
	if rcvState != state {
		return "", errors.New("oauth: state mismatch")
	}
	return rcvCode, nil
}

// Exchange swaps an auth code for a token at TokenURL. If TokenURL is
// empty, returns a mock token where AccessToken == code (local dev only).
func (f *Flow) Exchange(ctx context.Context, code string) (*Token, error) {
	if f.TokenURL == "" {
		return &Token{
			AccessToken: code,
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().Add(365 * 24 * time.Hour),
		}, nil
	}
	form := url.Values{
		"grant_type":   []string{"authorization_code"},
		"code":         []string{code},
		"redirect_uri": []string{fmt.Sprintf("http://127.0.0.1:%d/callback", f.Port)},
		"client_id":    []string{f.ClientID},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: token request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("oauth: read token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth: token endpoint %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("oauth: decode token response: %w", err)
	}
	if payload.AccessToken == "" {
		return nil, errors.New("oauth: token endpoint returned no access_token")
	}
	tt := payload.TokenType
	if tt == "" {
		tt = "Bearer"
	}
	expires := time.Now().Add(365 * 24 * time.Hour)
	if payload.ExpiresIn > 0 {
		expires = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	}
	return &Token{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		TokenType:    tt,
		ExpiresAt:    expires,
	}, nil
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

package invapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"
)

type TokenManager struct {
	mu          sync.RWMutex
	apiKey      string
	ttlSeconds  int
	token       string
	expiresAt   time.Time
	refreshLead time.Duration
	client      *Client
}

func NewTokenManager(apiKey string, ttlSeconds int, client *Client) *TokenManager {
	if ttlSeconds <= 0 {
		ttlSeconds = 3600
	}
	return &TokenManager{
		apiKey:      apiKey,
		ttlSeconds:  ttlSeconds,
		refreshLead: 2 * time.Minute,
		client:      client,
	}
}

func (tm *TokenManager) Token(ctx context.Context) (string, error) {
	tm.mu.RLock()
	token := tm.token
	expires := tm.expiresAt
	tm.mu.RUnlock()

	if token != "" && time.Now().Before(expires.Add(-tm.refreshLead)) {
		return token, nil
	}

	return tm.refresh(ctx)
}

// Invalidate clears the cached session token so the next Token() call re-logins.
func (tm *TokenManager) Invalidate() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.token = ""
	tm.expiresAt = time.Time{}
}

func (tm *TokenManager) refresh(ctx context.Context) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.token != "" && time.Now().Before(tm.expiresAt.Add(-tm.refreshLead)) {
		return tm.token, nil
	}

	params := url.Values{}
	params.Set("action", "login")
	params.Set("key", tm.apiKey)
	params.Set("ttl", fmt.Sprintf("%d", tm.ttlSeconds))

	body, err := tm.client.PostFormWithoutAuth(ctx, "auth", params)
	if err != nil {
		return "", fmt.Errorf("auth/login: %w", err)
	}

	var resp LoginResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("auth/login decode: %w", err)
	}

	token, invapiURL, _, expire := resp.Normalized()
	if token == "" {
		return "", fmt.Errorf("auth/login: empty token")
	}

	tm.token = token
	if expire > 0 {
		tm.expiresAt = time.Unix(expire, 0)
	} else {
		tm.expiresAt = time.Now().Add(time.Duration(tm.ttlSeconds) * time.Second)
	}

	if invapiURL != "" {
		canonical, err := CanonicalInvAPIBaseURL(invapiURL)
		current := tm.client.BaseURL()
		if err == nil && allowedInvAPIRewrite(current, canonical) == nil && current != canonical {
			tm.client.setBaseURL(canonical)
		}
	}

	return tm.token, nil
}

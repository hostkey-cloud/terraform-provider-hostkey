package invapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"
)

// ErrNoAppropriateServers is returned by auth/login when InvAPI refuses to
// issue a session because the account has zero servers. This is a known
// InvAPI limitation (chicken-and-egg with Terraform-first ordering), not an
// invalid API key. Auth docs only document key-based auth/login — there is
// no alternate empty-account session path the provider can use.
var ErrNoAppropriateServers = errors.New(
	"InvAPI auth/login refused: this account has no servers yet (NO_APPROPRIATE_SERVERS). " +
		"This is usually not a wrong API key — InvAPI will not create a session on an empty account, " +
		"so Terraform cannot order the first server either. Order the first server in the Hostkey panel " +
		"(or wait for an InvAPI fix), then re-run terraform",
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
		if IsNoAppropriateServers(err) {
			return "", ErrNoAppropriateServers
		}
		return "", fmt.Errorf("auth/login: %w", err)
	}

	var resp LoginResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("auth/login decode: %w", err)
	}

	token, invapiURL, _, expire := resp.Normalized()
	if token == "" {
		// Defensive: login envelopes with result=-1 normally fail in PostForm;
		// keep the same clear error if a future shape returns HTTP 200 + empty token.
		if IsNoAppropriateServers(errors.New(string(body))) {
			return "", ErrNoAppropriateServers
		}
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

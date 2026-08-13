package invapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type WaitOptions struct {
	PollInterval time.Duration
	Timeout      time.Duration
}

func (c *Client) CallbackCheck(ctx context.Context, key string) (*CallbackCheckResponse, error) {
	params := url.Values{}
	params.Set("action", "check")
	params.Set("key", key)

	body, err := c.PostForm(ctx, "eq_callback", params)
	if err != nil {
		return nil, fmt.Errorf("eq_callback/check: %w", err)
	}

	var resp CallbackCheckResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("eq_callback/check decode: %w", err)
	}
	return &resp, nil
}

func (c *Client) WaitForCallback(ctx context.Context, key string, opts WaitOptions) (*CallbackCheckResponse, error) {
	if key == "" {
		return nil, fmt.Errorf("callback key is empty")
	}

	interval := opts.PollInterval
	if interval <= 0 {
		interval = 15 * time.Second
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Minute
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		check, err := c.CallbackCheck(ctx, key)
		if err != nil {
			return nil, err
		}

		if done, err := callbackTerminal(check); done {
			if err != nil {
				return check, err
			}
			return check, nil
		}

		if time.Now().After(deadline) {
			return check, fmt.Errorf("callback %s timed out after %s", key, timeout)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func callbackTerminal(check *CallbackCheckResponse) (bool, error) {
	if check == nil {
		return false, nil
	}

	scope := strings.ToLower(string(check.Scope))
	if strings.Contains(scope, "deploy_done") || strings.Contains(scope, `"result":"deploy_done"`) {
		return true, nil
	}
	if strings.Contains(scope, "autodeploy completed") {
		return true, nil
	}

	if len(check.Context) > 0 {
		var ctx CallbackContext
		if err := json.Unmarshal(check.Context, &ctx); err == nil {
			if ctx.IP != "" && ctx.ID != "" {
				return true, nil
			}
		}

		ctxStr := strings.ToLower(string(check.Context))
		if strings.Contains(ctxStr, "error") {
			return true, fmt.Errorf("deploy failed: %s", string(check.Context))
		}
	}

	if strings.EqualFold(check.Result, "Error") {
		return true, fmt.Errorf("callback error: scope=%s debug=%s", check.Scope, check.Debug)
	}

	return false, nil
}

package invapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrPendingNotReady means this invoice/callback has no server id yet.
var ErrPendingNotReady = errors.New("pending deploy not ready")

func isPendingTerminalErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "deploy failed") ||
		strings.Contains(s, "callback error") ||
		strings.Contains(s, "pre-existing server")
}

func rejectKnownID(id, invoice int, known map[int]struct{}) error {
	if id <= 0 {
		return fmt.Errorf("invalid server id %d", id)
	}
	if _, existed := known[id]; existed {
		return fmt.Errorf("callback for invoice %d resolved to pre-existing server id %d", invoice, id)
	}
	return nil
}

func uniqueNewListID(known map[int]struct{}, ids []int) (int, error) {
	if len(known) == 0 && len(ids) > 1 {
		return 0, fmt.Errorf("missing pre-order snapshot; %d servers in eq/list, refusing to adopt an id", len(ids))
	}
	var newcomers []int
	for _, id := range ids {
		if _, ok := known[id]; !ok {
			newcomers = append(newcomers, id)
		}
	}
	switch len(newcomers) {
	case 0:
		return 0, ErrPendingNotReady
	case 1:
		return newcomers[0], nil
	default:
		return 0, fmt.Errorf("multiple new server ids %v; need invoice callback to disambiguate", newcomers)
	}
}

func newcomerIDs(known map[int]struct{}, ids []int) []int {
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if _, ok := known[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}

func showHostname(show *ServerShowResponse) string {
	if show == nil || len(show.ServerData) == 0 {
		return ""
	}
	var sd map[string]any
	if err := json.Unmarshal(show.ServerData, &sd); err != nil {
		return ""
	}
	for _, key := range []string{"hostname", "server_name", "name"} {
		if v, ok := sd[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (c *Client) matchPendingIDs(ctx context.Context, known map[int]struct{}, ids []int, wantHostname string) (int, error) {
	newcomers := newcomerIDs(known, ids)
	wantHostname = strings.TrimSpace(wantHostname)
	switch len(newcomers) {
	case 0:
		return 0, ErrPendingNotReady
	case 1:
		// A single newcomer after the pre-order snapshot is already safely
		// disambiguated. Do not require eq/show hostname matching here because
		// some panels lag or omit hostname fields even after the server exists.
		return newcomers[0], nil
	}

	if wantHostname == "" {
		return 0, fmt.Errorf("multiple new server ids %v; need invoice callback or hostname match to disambiguate", newcomers)
	}

	var matched []int
	for _, id := range newcomers {
		show, err := c.EQShow(ctx, id)
		if err != nil {
			continue
		}
		if strings.EqualFold(showHostname(show), wantHostname) {
			matched = append(matched, id)
		}
	}
	switch len(matched) {
	case 0:
		return 0, fmt.Errorf("multiple new server ids %v; none matched hostname %q", newcomers, wantHostname)
	case 1:
		return matched[0], nil
	default:
		return 0, fmt.Errorf("multiple new server ids %v matched hostname %q", matched, wantHostname)
	}
}

func (c *Client) matchPendingListID(ctx context.Context, known map[int]struct{}, wantHostname string) (int, error) {
	list, listErr := c.EQList(ctx, nil)
	if listErr != nil {
		return 0, listErr
	}
	ids, idErr := list.IDs()
	if idErr != nil {
		return 0, idErr
	}
	return c.matchPendingIDs(ctx, known, ids, wantHostname)
}

// LookupPendingServer is one poll for the server created by this invoice (and optional callback).
// When invoice > 0 it prefers deploy_keys/callback, but can safely fall back to
// eq/list when there is a single new server id or a hostname match disambiguates.
func (c *Client) LookupPendingServer(ctx context.Context, invoice int, callback string, known map[int]struct{}, wantHostname string) (id int, resolvedCallback string, err error) {
	if known == nil {
		known = map[int]struct{}{}
	}
	callback = strings.TrimSpace(callback)
	if callback == "" && invoice > 0 {
		upd, updErr := c.EQUpdateServers(ctx)
		if updErr != nil {
			return 0, "", updErr
		}
		callback = DeployKeyForInvoice(upd.DeployKeysMap(), invoice)
		if callback == "" && strings.TrimSpace(wantHostname) != "" {
			ids, idErr := upd.IDs()
			if idErr == nil {
				sid, matchErr := c.matchPendingIDs(ctx, known, ids, wantHostname)
				if matchErr == nil {
					return sid, "", nil
				}
				if !errors.Is(matchErr, ErrPendingNotReady) {
					// Keep going to eq/list fallback below; update_servers may lag or have a different shape.
				}
			}
		}
	}
	if callback != "" {
		check, cbErr := c.CallbackCheck(ctx, callback)
		if cbErr != nil {
			return 0, callback, cbErr
		}
		if _, termErr := callbackTerminal(check); termErr != nil {
			return 0, callback, termErr
		}
		sid := CallbackServerID(check)
		if sid == 0 {
			if invoice > 0 {
				sid, listErr := c.matchPendingListID(ctx, known, wantHostname)
				if listErr != nil {
					return 0, callback, listErr
				}
				return sid, callback, nil
			}
			return 0, callback, ErrPendingNotReady
		}
		if err := rejectKnownID(sid, invoice, known); err != nil {
			return 0, callback, err
		}
		return sid, callback, nil
	}
	if invoice > 0 {
		if strings.TrimSpace(wantHostname) == "" {
			return 0, "", ErrPendingNotReady
		}
		sid, listErr := c.matchPendingListID(ctx, known, wantHostname)
		if listErr != nil {
			return 0, "", listErr
		}
		return sid, "", nil
	}

	sid, matchErr := c.matchPendingListID(ctx, known, "")
	if matchErr != nil {
		return 0, "", matchErr
	}
	return sid, "", nil
}

// WaitForPendingServer polls LookupPendingServer until this invoice has a server id or timeout.
// Transient InvAPI/DNS errors are retried until Timeout; terminal deploy errors stop immediately.
func (c *Client) WaitForPendingServer(ctx context.Context, invoice int, callback string, known map[int]struct{}, wantHostname string, opts WaitOptions) (id int, resolvedCallback string, err error) {
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

	var lastErr error
	resolvedCallback = strings.TrimSpace(callback)

	for {
		sid, cb, lookErr := c.LookupPendingServer(ctx, invoice, resolvedCallback, known, wantHostname)
		if cb != "" {
			resolvedCallback = cb
		}
		if lookErr == nil && sid > 0 {
			return sid, resolvedCallback, nil
		}
		if lookErr != nil && !errors.Is(lookErr, ErrPendingNotReady) {
			lastErr = lookErr
			if isPendingTerminalErr(lookErr) {
				return 0, resolvedCallback, lookErr
			}
		} else {
			lastErr = ErrPendingNotReady
		}

		if time.Now().After(deadline) {
			if lastErr != nil && !errors.Is(lastErr, ErrPendingNotReady) {
				return 0, resolvedCallback, fmt.Errorf("timed out waiting for invoice %d after %s: %w", invoice, timeout, lastErr)
			}
			return 0, resolvedCallback, fmt.Errorf("timed out waiting for invoice %d after %s (deploy_keys/callback not ready)", invoice, timeout)
		}
		select {
		case <-ctx.Done():
			return 0, resolvedCallback, ctx.Err()
		case <-ticker.C:
		}
	}
}

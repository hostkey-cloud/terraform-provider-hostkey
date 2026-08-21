package invapi

import (
	"errors"
	"fmt"
	"strings"
)

// PendingTerminalError means this invoice/callback will never produce a server id
// (cancelled in the panel, deploy failed, async key purged, etc.). Waiters should
// stop immediately instead of polling until Timeout.
type PendingTerminalError struct {
	Message string
	Cause   error
}

func (e *PendingTerminalError) Error() string {
	if e == nil {
		return "pending deploy terminal failure"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "pending deploy terminal failure"
}

func (e *PendingTerminalError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func pendingTerminal(msg string, cause error) error {
	return &PendingTerminalError{Message: strings.TrimSpace(msg), Cause: cause}
}

// IsPendingTerminal reports whether err is a definitive cancel/fail for a pending
// deploy (including legacy string-matched messages from older callbackTerminal paths).
func IsPendingTerminal(err error) bool {
	if err == nil {
		return false
	}
	var term *PendingTerminalError
	if errors.As(err, &term) {
		return true
	}
	return isPendingTerminalErr(err)
}

func isPendingTerminalErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "deploy failed") ||
		strings.Contains(s, "deploy cancelled") ||
		strings.Contains(s, "callback error") ||
		strings.Contains(s, "pre-existing server")
}

// isCallbackKeyGone reports whether CallbackCheck failed because the async key
// no longer exists (typical after panel cancel / failed deploy cleanup).
func isCallbackKeyGone(err error) bool {
	if err == nil {
		return false
	}
	if IsNotFound(err) {
		return true
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "not found") || strings.Contains(s, "does not exist") {
		return strings.Contains(s, "asynckey") ||
			strings.Contains(s, "async key") ||
			strings.Contains(s, "callback") ||
			strings.Contains(s, "deploy key")
	}
	return false
}

func callbackGoneTerminal(callback string, cause error) error {
	cb := strings.TrimSpace(callback)
	msg := "deploy cancelled or failed: async callback key no longer exists"
	if cb != "" {
		msg = fmt.Sprintf("%s (callback=%q)", msg, cb)
	}
	if cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, cause)
	}
	return pendingTerminal(msg, cause)
}

// terminalFromCallbackCheckErr maps CallbackCheck transport/API failures that
// already mean a definitive cancel/fail. PostForm rejects non-OK "result" before
// callbackTerminal can see the JSON body, so Result=Error arrives here as APIError.
func terminalFromCallbackCheckErr(err error) error {
	if err == nil {
		return nil
	}
	var api *APIError
	if errors.As(err, &api) {
		result := strings.ToLower(strings.TrimSpace(api.Result))
		switch result {
		case "error", "failed", "cancelled", "canceled", "fail":
			return pendingTerminal(fmt.Sprintf("callback error: result=%s", api.Result), err)
		}
		blob := strings.ToLower(strings.TrimSpace(api.Message + " " + api.Body + " " + api.Result))
		if reason := terminalFailReason(blob); reason != "" {
			return pendingTerminal(fmt.Sprintf("deploy failed (%s): %v", reason, err), err)
		}
	}
	blob := strings.ToLower(err.Error())
	if reason := terminalFailReason(blob); reason != "" &&
		(strings.Contains(blob, "result=error") ||
			strings.Contains(blob, "result=failed") ||
			strings.Contains(blob, "result=cancelled") ||
			strings.Contains(blob, "result=canceled")) {
		return pendingTerminal(fmt.Sprintf("callback error: %v", err), err)
	}
	return nil
}

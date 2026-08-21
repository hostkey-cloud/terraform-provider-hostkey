package invapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCallbackTerminal_ErrorResult(t *testing.T) {
	done, err := callbackTerminal(&CallbackCheckResponse{
		Result: "Error",
		Scope:  []byte(`"pending"`),
		Debug:  "cancelled by user",
	})
	if !done || err == nil || !IsPendingTerminal(err) {
		t.Fatalf("want terminal error, got done=%v err=%v", done, err)
	}
	if !strings.Contains(err.Error(), "callback error") {
		t.Fatalf("unexpected message: %v", err)
	}
}

func TestCallbackTerminal_ContextCancelled(t *testing.T) {
	done, err := callbackTerminal(&CallbackCheckResponse{
		Result:  "OK",
		Context: []byte(`{"action":"deploy","status":"cancelled"}`),
	})
	if !done || !IsPendingTerminal(err) {
		t.Fatalf("want terminal cancel, got done=%v err=%v", done, err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "cancelled") {
		t.Fatalf("unexpected message: %v", err)
	}
}

func TestCallbackTerminal_ContextError(t *testing.T) {
	done, err := callbackTerminal(&CallbackCheckResponse{
		Result:  "OK",
		Context: []byte(`{"error":"deploy aborted by panel"}`),
	})
	if !done || !IsPendingTerminal(err) {
		t.Fatalf("want terminal fail, got done=%v err=%v", done, err)
	}
}

func TestCallbackTerminal_SuccessStillWins(t *testing.T) {
	done, err := callbackTerminal(&CallbackCheckResponse{
		Result:  "OK",
		Scope:   []byte(`"deploy_done"`),
		Context: []byte(`{"id":"101","ip":"1.2.3.4"}`),
	})
	if !done || err != nil {
		t.Fatalf("want success done, got done=%v err=%v", done, err)
	}
}

func TestWaitForPendingServer_StopsOnCallbackError(t *testing.T) {
	freshPendingClaims(t)
	var polls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		polls++
		switch {
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"result":"Error","scope":"cancelled","debug":"order cancelled in panel","key":"cb-ours"}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"603548":"cb-ours"},"servers":[10]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, waitErr := c.WaitForPendingServer(context.Background(), 603548, "cb-ours", map[int]struct{}{10: {}}, "tf-host", WaitOptions{
		PollInterval: 20 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if !IsPendingTerminal(waitErr) {
		t.Fatalf("want terminal, got %v", waitErr)
	}
	if polls > 5 {
		t.Fatalf("expected immediate stop, polls=%d", polls)
	}
}

func TestWaitForPendingServer_StopsOnAsyncKeyGone(t *testing.T) {
	freshPendingClaims(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"code":-1,"message":"AsyncKey cb-gone not found"}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,20]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":[],"servers":[10,20]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, waitErr := c.WaitForPendingServer(context.Background(), 603548, "cb-gone", map[int]struct{}{10: {}, 20: {}}, "tf-missing", WaitOptions{
		PollInterval: 20 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if !IsPendingTerminal(waitErr) {
		t.Fatalf("want terminal key-gone, got %v", waitErr)
	}
	if !strings.Contains(strings.ToLower(waitErr.Error()), "callback key no longer exists") {
		t.Fatalf("unexpected message: %v", waitErr)
	}
}

func TestLookupPendingServer_AsyncKeyGoneLinksExistingNewcomer(t *testing.T) {
	freshPendingClaims(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"code":-1,"message":"AsyncKey cb-done not found"}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,101]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show":
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{"hostname":"tf-linked-host"}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, _, lookErr := c.LookupPendingServer(context.Background(), 603548, "cb-done", map[int]struct{}{10: {}}, "tf-linked-host")
	if lookErr != nil || id != 101 {
		t.Fatalf("want link via hostname after key gone, got id=%d err=%v", id, lookErr)
	}
}

func TestIsPendingTerminal_TypedAndLegacy(t *testing.T) {
	if !IsPendingTerminal(pendingTerminal("deploy cancelled or failed: x", nil)) {
		t.Fatal("typed")
	}
	if !IsPendingTerminal(errors.New("callback error: scope=x")) {
		t.Fatal("legacy callback error")
	}
	if IsPendingTerminal(ErrPendingNotReady) {
		t.Fatal("not-ready must not be terminal")
	}
}

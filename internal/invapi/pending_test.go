package invapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeployKeyForInvoice(t *testing.T) {
	keys := map[string]string{"603548": "cb-a", "111": "cb-b"}
	if got := DeployKeyForInvoice(keys, 603548); got != "cb-a" {
		t.Fatalf("got %q", got)
	}
	if got := DeployKeyForInvoice(keys, 999); got != "" {
		t.Fatalf("missing key: %q", got)
	}
	if got := DeployKeyForInvoice(nil, 603548); got != "" {
		t.Fatalf("nil map: %q", got)
	}
}

func TestUniqueNewListID(t *testing.T) {
	known := map[int]struct{}{10: {}, 20: {}}
	id, err := uniqueNewListID(known, []int{10, 20, 101})
	if err != nil || id != 101 {
		t.Fatalf("got %d %v", id, err)
	}
	_, err = uniqueNewListID(known, []int{10, 20, 101, 102})
	if err == nil || !strings.Contains(err.Error(), "multiple new") {
		t.Fatalf("expected multiple new, got %v", err)
	}
	_, err = uniqueNewListID(map[int]struct{}{}, []int{10, 20})
	if err == nil || !strings.Contains(err.Error(), "missing pre-order snapshot") {
		t.Fatalf("expected fail-closed empty known, got %v", err)
	}
	id, err = uniqueNewListID(map[int]struct{}{}, []int{101})
	if err != nil || id != 101 {
		t.Fatalf("empty account first server: %d %v", id, err)
	}
}

func TestWaitForPendingServer_BindsInvoiceCallback(t *testing.T) {
	var listed atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"603548":"cb-ours","111":"cb-other"}}`)
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			key := r.Form.Get("key")
			if key == "cb-ours" {
				_, _ = io.WriteString(w, `{"result":"OK","scope":"deploy_done","context":{"id":"101","ip":"1.2.3.4"}}`)
				return
			}
			_, _ = io.WriteString(w, `{"result":"OK","scope":"deploy_done","context":{"id":"100","ip":"9.9.9.9"}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			listed.Add(1)
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,20,100,101]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	known := map[int]struct{}{10: {}, 20: {}}
	id, cb, err := c.WaitForPendingServer(context.Background(), 603548, "", known, "", WaitOptions{
		PollInterval: 10 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != 101 {
		t.Fatalf("id=%d want 101", id)
	}
	if cb != "cb-ours" {
		t.Fatalf("callback=%q", cb)
	}
	if listed.Load() != 0 {
		t.Fatalf("eq/list should not run when invoice is set, hits=%d", listed.Load())
	}
}

func TestWaitForPendingServer_DoesNotAdoptForeignListID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"111":"cb-other"}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,100]}`)
		default:
			_, _ = io.WriteString(w, `{"result":"OK"}`)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = c.WaitForPendingServer(context.Background(), 603548, "", map[int]struct{}{10: {}}, "", WaitOptions{
		PollInterval: 20 * time.Millisecond,
		Timeout:      80 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout, not a foreign server id")
	}
	if !strings.Contains(err.Error(), "timed out waiting for invoice 603548") {
		t.Fatalf("err=%v", err)
	}
}

func TestWaitForPendingServer_RetriesUpdateServersError(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers" {
			n := hits.Add(1)
			if n == 1 {
				w.WriteHeader(http.StatusBadGateway)
				_, _ = io.WriteString(w, "bad gateway")
				return
			}
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"222":"cb-ok"}}`)
			return
		}
		if strings.Contains(r.URL.Path, "eq_callback.php") {
			_, _ = io.WriteString(w, `{"result":"OK","scope":"deploy_done","context":{"id":"50","ip":"10.0.0.1"}}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := c.WaitForPendingServer(context.Background(), 222, "", map[int]struct{}{}, "", WaitOptions{
		PollInterval: 20 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != 50 {
		t.Fatalf("id=%d", id)
	}
	if hits.Load() < 2 {
		t.Fatalf("expected retry after update_servers error, hits=%d", hits.Load())
	}
}

func TestLookupPendingServer_NotReady(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":[]}`)
	}))
	defer srv.Close()
	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := c.LookupPendingServer(context.Background(), 603548, "", map[int]struct{}{1: {}}, "")
	if !errors.Is(err, ErrPendingNotReady) {
		t.Fatalf("got id=%d err=%v", id, err)
	}
}

func TestWaitForPendingServer_FallsBackToSingleNewListID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"603548":"cb-ours"}}`)
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"result":"OK","scope":"pending","context":{"id":"","ip":""}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,20,101]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := c.WaitForPendingServer(context.Background(), 603548, "", map[int]struct{}{10: {}, 20: {}}, "", WaitOptions{
		PollInterval: 10 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != 101 {
		t.Fatalf("id=%d want 101", id)
	}
}

func TestWaitForPendingServer_HostnameDisambiguatesMultipleNewIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"603548":"cb-ours"}}`)
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"result":"OK","scope":"pending","context":{"id":"","ip":""}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,101,102]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show" && r.Form.Get("id") == "101":
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{"hostname":"other-host"}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show" && r.Form.Get("id") == "102":
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{"hostname":"tf-pico-renamed"}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := c.WaitForPendingServer(context.Background(), 603548, "", map[int]struct{}{10: {}}, "tf-pico-renamed", WaitOptions{
		PollInterval: 10 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != 102 {
		t.Fatalf("id=%d want 102", id)
	}
}

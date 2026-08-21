package invapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func freshPendingClaims(t *testing.T) {
	t.Helper()
	resetPendingServerClaimsForTest()
	t.Cleanup(resetPendingServerClaimsForTest)
}

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

func TestPendingClaim_SameOwnerReclaims(t *testing.T) {
	freshPendingClaims(t)
	if !tryClaimPendingServerID(101, "invoice:1") {
		t.Fatal("first claim")
	}
	if !tryClaimPendingServerID(101, "invoice:1") {
		t.Fatal("same owner must reclaim")
	}
	if tryClaimPendingServerID(101, "invoice:2") {
		t.Fatal("other owner must not steal")
	}
	ReleasePendingServerClaim(101, "invoice:1")
	if !tryClaimPendingServerID(101, "invoice:2") {
		t.Fatal("after release other owner may claim")
	}
}

func TestWaitForPendingServer_BindsInvoiceCallback(t *testing.T) {
	freshPendingClaims(t)
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
	freshPendingClaims(t)
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
	freshPendingClaims(t)
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
	freshPendingClaims(t)
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

func TestCallbackServerID_UsesContextOrScopeID(t *testing.T) {
	if got := CallbackServerID(&CallbackCheckResponse{
		Context: json.RawMessage(`{"id":"12345","ip":"1.2.3.4"}`),
	}); got != 12345 {
		t.Fatalf("context id: got %d", got)
	}

	if got := CallbackServerID(&CallbackCheckResponse{
		Scope: json.RawMessage(`{"id":67890,"location":"RU"}`),
	}); got != 67890 {
		t.Fatalf("scope id: got %d", got)
	}
}

func TestShowHostname_ExtractsNestedHostname(t *testing.T) {
	show := &ServerShowResponse{
		ServerData: json.RawMessage(`{
			"data": {
				"server": {
					"hostname": "tf-pico-renamed"
				}
			}
		}`),
	}
	if got := showHostname(show); got != "tf-pico-renamed" {
		t.Fatalf("got %q want %q", got, "tf-pico-renamed")
	}
}

func TestShowHostname_IgnoresNestedUnrelatedNameField(t *testing.T) {
	show := &ServerShowResponse{
		ServerData: json.RawMessage(`{
			"data": {
				"preset": {
					"name": "vps.v1.small"
				}
			}
		}`),
	}
	if got := showHostname(show); got != "" {
		t.Fatalf("got %q, want empty (nested preset name must not be treated as hostname)", got)
	}
}

func TestShowHostname_TrustsTopLevelNameField(t *testing.T) {
	show := &ServerShowResponse{
		ServerData: json.RawMessage(`{"name": "tf-pico-renamed", "preset": {"name": "vps.v1.small"}}`),
	}
	if got := showHostname(show); got != "tf-pico-renamed" {
		t.Fatalf("got %q want %q", got, "tf-pico-renamed")
	}
}

func TestShowHostname_FromTagsHostname(t *testing.T) {
	show := &ServerShowResponse{
		ServerData: json.RawMessage(`{"status":"Active"}`),
		Tags:       json.RawMessage(`[{"tag":"hostname","value":"tf-3vm-fi-120711"},{"tag":"os","value":"Debian 12"}]`),
	}
	if got := showHostname(show); got != "tf-3vm-fi-120711" {
		t.Fatalf("got %q want %q", got, "tf-3vm-fi-120711")
	}
}

func TestShowHostname_FromTagsServerNameWhenNoHostname(t *testing.T) {
	show := &ServerShowResponse{
		Tags: json.RawMessage(`[{"tag":"server_name","value":"my-server"},{"tag":"os","value":"Ubuntu"}]`),
	}
	if got := showHostname(show); got != "my-server" {
		t.Fatalf("got %q want %q", got, "my-server")
	}
}

func TestShowHostname_PrefersServerDataOverTags(t *testing.T) {
	show := &ServerShowResponse{
		ServerData: json.RawMessage(`{"hostname":"from-server-data"}`),
		Tags:       json.RawMessage(`[{"tag":"hostname","value":"from-tags"}]`),
	}
	if got := showHostname(show); got != "from-server-data" {
		t.Fatalf("got %q want %q", got, "from-server-data")
	}
}

func TestShowHostname_TagsOnlyWhenServerDataEmpty(t *testing.T) {
	show := &ServerShowResponse{
		Tags: json.RawMessage(`[{"tag":"hostname","value":"tags-only-host"}]`),
	}
	if got := showHostname(show); got != "tags-only-host" {
		t.Fatalf("got %q want %q", got, "tags-only-host")
	}
}

func TestShowContainsHostname_MatchesTags(t *testing.T) {
	show := &ServerShowResponse{
		ServerData: json.RawMessage(`{}`),
		Tags:       json.RawMessage(`[{"tag":"hostname","value":"tf-pico-renamed"}]`),
	}
	if !showContainsHostname(show, "tf-pico-renamed") {
		t.Fatalf("expected tags hostname match")
	}
	if showContainsHostname(show, "other-host") {
		t.Fatalf("expected no match for other-host")
	}
}

func TestShowContainsHostname_FindsNestedStringValue(t *testing.T) {
	show := &ServerShowResponse{
		ServerData: json.RawMessage(`{
			"config": {
				"some": {
					"nested_value": "tf-pico-renamed"
				}
			}
		}`),
	}
	if !showContainsHostname(show, "tf-pico-renamed") {
		t.Fatalf("expected hostname to be found")
	}
}

func TestWaitForPendingServer_FallsBackToSingleNewListID(t *testing.T) {
	freshPendingClaims(t)
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

func TestWaitForPendingServer_SingleNewcomerLinksWithoutWantHostname(t *testing.T) {
	freshPendingClaims(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"603548":"cb-ours"},"servers":[10,20,101]}`)
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"result":"OK","scope":"pending","context":{"id":"","ip":""}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,20,101]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show":
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Empty wantHostname: single newcomer still links even when eq/show has no hostname.
	id, _, err := c.WaitForPendingServer(context.Background(), 603548, "", map[int]struct{}{10: {}, 20: {}}, "", WaitOptions{
		PollInterval: 10 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatalf("expected single newcomer link without wantHostname, got err=%v", err)
	}
	if id != 101 {
		t.Fatalf("id=%d want 101", id)
	}
}

func TestWaitForPendingServer_WantHostnameSkipsEmptySingleNewcomer(t *testing.T) {
	freshPendingClaims(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"603548":"cb-ours"},"servers":[10,20,101]}`)
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"result":"OK","scope":"pending","context":{"id":"","ip":""}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,20,101]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show":
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = c.WaitForPendingServer(context.Background(), 603548, "", map[int]struct{}{10: {}, 20: {}}, "tf-pending-want-host", WaitOptions{
		PollInterval: 10 * time.Millisecond,
		Timeout:      200 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout while waiting for hostname match, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestWaitForPendingServer_SingleNewcomerLinksViaInvAPIDefaultHostname(t *testing.T) {
	freshPendingClaims(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"603548":"cb-ours"},"servers":[10,20,101]}`)
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"result":"OK","scope":"pending","context":{"id":"","ip":""}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,20,101]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show":
			// InvAPI default before requested hostname is applied.
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{"status":"Active"},"tags":[{"tag":"hostname","value":"hostkey101"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := c.WaitForPendingServer(context.Background(), 603548, "", map[int]struct{}{10: {}, 20: {}}, "tf-pending-want-host", WaitOptions{
		PollInterval: 10 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatalf("expected hostkey{id} placeholder to allow single-newcomer claim, got err=%v", err)
	}
	if id != 101 {
		t.Fatalf("id=%d want 101", id)
	}
}

func TestWaitForPendingServer_SingleNewcomerLinksViaTagsHostname(t *testing.T) {
	freshPendingClaims(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"603548":"cb-ours"},"servers":[10,20,101]}`)
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"result":"OK","scope":"pending","context":{"id":"","ip":""}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,20,101]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show":
			// Live InvAPI often puts hostname only in tags[], not server_data.
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{"status":"Active"},"tags":[{"tag":"hostname","value":"tf-pending-tags-host"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := c.WaitForPendingServer(context.Background(), 603548, "", map[int]struct{}{10: {}, 20: {}}, "tf-pending-tags-host", WaitOptions{
		PollInterval: 10 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatalf("expected tags hostname match to link, got err=%v", err)
	}
	if id != 101 {
		t.Fatalf("id=%d want 101", id)
	}
}

func TestWaitForPendingServer_ConcurrentWaitersMatchHostnameNotFirstNewcomer(t *testing.T) {
	freshPendingClaims(t)

	// Parallel Creates with wantHostname set must not claim the first empty-hostname
	// newcomer; each waiter waits until tags/server_data hostname matches.
	var publishHostnames atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"1001":"cb-a","1002":"cb-b"},"servers":[10,101,102]}`)
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"result":"OK","scope":"pending","context":{"id":"","ip":""}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,101,102]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show":
			id := r.Form.Get("id")
			if !publishHostnames.Load() {
				_, _ = io.WriteString(w, `{"result":"OK","server_data":{}}`)
				return
			}
			switch id {
			case "101":
				_, _ = io.WriteString(w, `{"result":"OK","server_data":{},"tags":[{"tag":"hostname","value":"host-a"}]}`)
			case "102":
				_, _ = io.WriteString(w, `{"result":"OK","server_data":{},"tags":[{"tag":"hostname","value":"host-b"}]}`)
			default:
				_, _ = io.WriteString(w, `{"result":"OK","server_data":{}}`)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	known := map[int]struct{}{10: {}}
	opts := WaitOptions{PollInterval: 15 * time.Millisecond, Timeout: 3 * time.Second}

	type result struct {
		invoice int
		id      int
		err     error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	wantByInvoice := map[int]string{1001: "host-a", 1002: "host-b"}
	for _, invoice := range []int{1001, 1002} {
		invoice := invoice
		go func() {
			defer wg.Done()
			id, _, waitErr := c.WaitForPendingServer(context.Background(), invoice, "", known, wantByInvoice[invoice], opts)
			results <- result{invoice: invoice, id: id, err: waitErr}
		}()
	}

	// Brief empty-hostname window (would previously cause cross-link), then publish tags.
	time.Sleep(40 * time.Millisecond)
	publishHostnames.Store(true)

	wg.Wait()
	close(results)

	got := map[int]int{}
	for r := range results {
		if r.err != nil {
			t.Fatalf("invoice %d: %v", r.invoice, r.err)
		}
		got[r.invoice] = r.id
	}
	if got[1001] != 101 {
		t.Fatalf("invoice 1001: id=%d want 101", got[1001])
	}
	if got[1002] != 102 {
		t.Fatalf("invoice 1002: id=%d want 102", got[1002])
	}
}

func TestLookupPendingServer_WantHostnameEmptySingleNewcomerNotClaimed(t *testing.T) {
	freshPendingClaims(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"2001":"cb-a","2002":"cb-b"},"servers":[10,201]}`)
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			_, _ = io.WriteString(w, `{"result":"OK","scope":"pending","context":{"id":"","ip":""}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show":
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	known := map[int]struct{}{10: {}}

	var wg sync.WaitGroup
	var wins atomic.Int32
	var ready atomic.Int32
	start := make(chan struct{})
	wg.Add(2)
	for _, invoice := range []int{2001, 2002} {
		invoice := invoice
		go func() {
			defer wg.Done()
			ready.Add(1)
			<-start
			id, _, lookErr := c.LookupPendingServer(context.Background(), invoice, "", known, "lagging-hostname")
			if lookErr == nil && id == 201 {
				wins.Add(1)
			} else if lookErr == nil {
				t.Errorf("invoice %d got unexpected id %d", invoice, id)
			} else if !errors.Is(lookErr, ErrPendingNotReady) {
				t.Errorf("invoice %d: %v", invoice, lookErr)
			}
		}()
	}
	for ready.Load() < 2 {
		time.Sleep(time.Millisecond)
	}
	close(start)
	wg.Wait()
	if wins.Load() != 0 {
		t.Fatalf("expected no waiter to claim empty-hostname newcomer when wantHostname set, got %d wins", wins.Load())
	}
}

func TestWaitForPendingServer_CallbackSidZeroUsesUpdateServersIDsFirst(t *testing.T) {
	freshPendingClaims(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq_callback.php"):
			// Callback is still pending; does not carry a server id yet.
			_, _ = io.WriteString(w, `{"result":"OK","scope":"pending","context":{"id":"","ip":""}}`)
			return
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			// update_servers shows exactly one new server candidate (101).
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":{"603548":"cb-ours"},"servers":[10,101],"billing_servers":[]}`)
			return
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			// eq/list shows two new ids, and would normally require hostname disambiguation.
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,101,102]}`)
			return
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show" && r.Form.Get("id") == "101":
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{"hostname":"tf-pending-timeoutcheck-20260819-1846"}}`)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}

	known := map[int]struct{}{10: {}}
	id, _, err := c.WaitForPendingServer(context.Background(), 603548, "cb-ours", known, "tf-pending-timeoutcheck-20260819-1846", WaitOptions{
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
	freshPendingClaims(t)
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

func TestWaitForPendingServer_NoCallbackFallsBackByHostname(t *testing.T) {
	freshPendingClaims(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":[]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			_, _ = io.WriteString(w, `{"result":"OK","servers":[10,101]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show" && r.Form.Get("id") == "101":
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{"hostname":"tf-pending-fix-20260819-1425"}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), MaxRetries: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := c.WaitForPendingServer(context.Background(), 603548, "", map[int]struct{}{10: {}}, "tf-pending-fix-20260819-1425", WaitOptions{
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

func TestWaitForPendingServer_NoCallbackUsesUpdateServersListFirst(t *testing.T) {
	freshPendingClaims(t)
	var listHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "update_servers":
			_, _ = io.WriteString(w, `{"result":"OK","deploy_keys":[],"servers":[10,101]}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "show" && r.Form.Get("id") == "101":
			_, _ = io.WriteString(w, `{"result":"OK","server_data":{"hostname":"tf-pending-fix-20260819-1504"}}`)
		case strings.Contains(r.URL.Path, "eq.php") && r.Form.Get("action") == "list":
			listHits.Add(1)
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
	id, _, err := c.WaitForPendingServer(context.Background(), 603548, "", map[int]struct{}{10: {}}, "tf-pending-fix-20260819-1504", WaitOptions{
		PollInterval: 10 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != 101 {
		t.Fatalf("id=%d want 101", id)
	}
	if listHits.Load() != 0 {
		t.Fatalf("eq/list should not be needed when update_servers already exposes the new server, hits=%d", listHits.Load())
	}
}

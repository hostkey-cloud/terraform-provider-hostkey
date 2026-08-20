package invapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDecodeAPIError_OK(t *testing.T) {
	if err := decodeAPIError([]byte(`{"result":"OK"}`)); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestDecodeAPIError_Message(t *testing.T) {
	err := decodeAPIError([]byte(`{"code":1,"error":"boom"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	api, ok := err.(*APIError)
	if !ok || api.Message != "boom" {
		t.Fatalf("got %#v", err)
	}
}

func TestPostForm_Retries500(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, "bad gateway")
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"result":"OK"}`)
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:    srv.URL + "/",
		MaxRetries: 4,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		UserAgent:  "test-ua/1",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	body, err := client.PostFormWithoutAuth(context.Background(), "eq", url.Values{"action": {"ping"}})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !strings.Contains(string(body), "OK") {
		t.Fatalf("body=%s", body)
	}
	if hits.Load() != 3 {
		t.Fatalf("hits=%d", hits.Load())
	}
}

func TestPostForm_NoRetryOrderInstance(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "bad gateway")
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:    srv.URL + "/",
		MaxRetries: 4,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.PostFormWithoutAuth(context.Background(), "eq", url.Values{
		"action":    {"order_instance"},
		"root_pass": {"Secret1+"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if hits.Load() != 1 {
		t.Fatalf("order_instance must not retry on 502, hits=%d", hits.Load())
	}
}

func TestIsNonRetryableForm(t *testing.T) {
	nonRetryable := []struct{ module, action string }{
		{"eq", "order_instance"},
		{"net", "add_ipv4"},
		{"pdns", "add_domain"},
		{"pdns", "add_dns"},
		{"ssh_keys", "add"},
		{"tags", "add"},
	}
	for _, tc := range nonRetryable {
		if !isNonRetryableForm(tc.module, url.Values{"action": {tc.action}}) {
			t.Fatalf("%s/%s must be non-retryable", tc.module, tc.action)
		}
	}

	retryable := []struct{ module, action string }{
		{"eq", "show"},
		{"eq", "list"},
		{"eq", "rename_server"},
		{"eq", "on"},
		{"eq", "off"},
		{"eq", "update_servers"},
		{"pdns", "delete_dns"},
		{"pdns", "delete_domain"},
		{"pdns", "view_zone"},
		{"net", "remove_ipv4"},
		{"ssh_keys", "delete"},
		{"ssh_keys", "list"},
		{"tags", "remove"},
		{"tags", "list"},
		{"presets", "list"},
	}
	for _, tc := range retryable {
		if isNonRetryableForm(tc.module, url.Values{"action": {tc.action}}) {
			t.Fatalf("%s/%s must be retryable", tc.module, tc.action)
		}
	}
}

func TestPostForm_SetsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "terraform-provider-hostkey/test" {
			t.Errorf("ua=%q", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"result":"OK"}`)
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:   srv.URL + "/",
		UserAgent: "terraform-provider-hostkey/test",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.PostFormWithoutAuth(context.Background(), "eq", url.Values{}); err != nil {
		t.Fatal(err)
	}
}

func TestPostForm_AuthRefreshOn401(t *testing.T) {
	var authHits, eqHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch {
		case strings.Contains(r.URL.Path, "auth.php"):
			authHits.Add(1)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, fmt.Sprintf(`{"token":"tok-%d","token_expire":9999999999}`, authHits.Load()))
		case strings.Contains(r.URL.Path, "eq.php"):
			n := eqHits.Add(1)
			if n == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `unauthorized`)
				return
			}
			if got := r.Form.Get("token"); got != "tok-2" {
				t.Errorf("expected refreshed token, got %q", got)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"result":"OK"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:    srv.URL + "/",
		MaxRetries: 3,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	auth := NewTokenManager("key", 3600, client)
	client.SetAuth(auth)

	if _, err := client.PostForm(context.Background(), "eq", url.Values{"action": {"show"}}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if authHits.Load() < 2 {
		t.Fatalf("expected re-login, authHits=%d", authHits.Load())
	}
}

func TestIsAuthFailure(t *testing.T) {
	if !isAuthFailure(401, nil) {
		t.Fatal("401")
	}
	if isAuthFailure(403, nil) {
		t.Fatal("403 should not force auth retry alone")
	}
	if !isAuthFailure(200, &APIError{Message: "Invalid token"}) {
		t.Fatal("message")
	}
}

func TestCheckRedirect_BlocksCrossOrigin(t *testing.T) {
	evilHits := atomic.Int32{}
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		evilHits.Add(1)
		_, _ = io.WriteString(w, `{"result":"OK"}`)
	}))
	defer evil.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+"/steal", http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	client, err := NewClient(Config{
		BaseURL:    srv.URL + "/",
		MaxRetries: 1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.PostFormWithoutAuth(context.Background(), "eq", url.Values{"action": {"show"}})
	if err == nil || !strings.Contains(err.Error(), "cross-origin redirect") {
		t.Fatalf("expected cross-origin redirect error, got %v", err)
	}
	if evilHits.Load() != 0 {
		t.Fatal("POST body must not follow 307 to another origin")
	}
}

func TestLogin_IgnoresForeignInvapi(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"token":"sess","invapi":"https://evil.example/"}`)
	}))
	defer srv.Close()

	client, err := NewClient(Config{BaseURL: srv.URL + "/", MaxRetries: 1, HTTPClient: srv.Client()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	auth := NewTokenManager("key", 3600, client)
	client.SetAuth(auth)
	if _, err := auth.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := client.BaseURL(); !strings.HasPrefix(got, srv.URL) {
		t.Fatalf("baseURL rewritten to %q", got)
	}
}

func TestRetryableStatus(t *testing.T) {
	for _, code := range []int{0, 429, 500, 502, 503, 504} {
		if !retryableStatus(code) {
			t.Fatalf("%d", code)
		}
	}
	if retryableStatus(400) {
		t.Fatal("400")
	}
}

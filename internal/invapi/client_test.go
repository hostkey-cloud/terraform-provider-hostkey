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

package invapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPDNSDeleteDNS_SendsContent(t *testing.T) {
	var captured url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		captured = r.PostForm
		_, _ = io.WriteString(w, `{"result":"OK"}`)
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = c.PDNSDeleteDNS(context.Background(), PDNSDeleteDNSRequest{
		Zone:     "example.com",
		Name:     "@",
		Type:     "NS",
		Content:  "ns1.example.net.",
		Priority: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Get("action") != "delete_dns" {
		t.Fatalf("action=%q", captured.Get("action"))
	}
	if captured.Get("params[type]") != "NS" {
		t.Fatalf("type=%q", captured.Get("params[type]"))
	}
	contents := captured["params[content][]"]
	if len(contents) != 1 || contents[0] != "ns1.example.net." {
		t.Fatalf("content=%v", contents)
	}
	if captured.Get("params[priority]") != "10" {
		t.Fatalf("priority=%q", captured.Get("params[priority]"))
	}
}

func TestPDNSDeleteDNS_RequiresContent(t *testing.T) {
	c, err := NewClient(Config{BaseURL: "https://example.com/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = c.PDNSDeleteDNS(context.Background(), PDNSDeleteDNSRequest{
		Zone: "example.com",
		Name: "@",
		Type: "NS",
	})
	if err == nil || !strings.Contains(err.Error(), "content is required") {
		t.Fatalf("err=%v", err)
	}
}

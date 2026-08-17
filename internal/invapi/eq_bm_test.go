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

func TestEQOrderInstanceBareMetalParams(t *testing.T) {
	var captured url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		captured = r.PostForm
		_, _ = io.WriteString(w, `{"result":"OK","action":"order_instance"}`)
	}))
	defer srv.Close()

	noLVM := true
	ipv6 := true
	c, err := NewClient(Config{BaseURL: srv.URL + "/"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.EQOrderInstance(context.Background(), OrderInstanceRequest{
		Preset:       "bm.v2-promo",
		LocationName: "NL",
		RootPass:     "Abcdef1%",
		OSID:         187,
		DiskMirror:   "raid1",
		NoLVM:        &noLVM,
		IPv6Block:    &ipv6,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := captured.Get("disk_mirror"); got != "raid1" {
		t.Fatalf("disk_mirror: %q", got)
	}
	if got := captured.Get("no_lvm"); got != "1" {
		t.Fatalf("no_lvm: %q", got)
	}
	if got := captured.Get("ipv6"); got != "1" {
		t.Fatalf("ipv6: %q", got)
	}
	if !strings.HasSuffix(captured.Get("preset"), "bm.v2-promo") && captured.Get("preset") != "bm.v2-promo" {
		t.Fatalf("preset: %q", captured.Get("preset"))
	}
}

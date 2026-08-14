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

func TestEQOrderInstanceDropsReservedExtra(t *testing.T) {
	var captured url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		captured = r.PostForm
		_, _ = io.WriteString(w, `{"result":"OK","action":"order_instance"}`)
	}))
	defer srv.Close()

	c, err := NewClient(Config{BaseURL: srv.URL + "/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.EQOrderInstance(context.Background(), OrderInstanceRequest{
		Preset:   "vm.pico",
		RootPass: "Abcdef1%",
		Extra: map[string]string{
			"price": "1",
			"id":    "99999",
			"token": "stolen",
			"uefi":  "1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Get("price") != "" {
		t.Fatal("price must not be forwarded from Extra")
	}
	if captured.Get("id") != "" {
		t.Fatal("id must not be forwarded from Extra on create")
	}
	if captured.Get("uefi") != "" {
		t.Fatalf("extra_order_params must not be forwarded; got uefi=%q", captured.Get("uefi"))
	}
}

func TestIsReservedOrderExtraKey(t *testing.T) {
	if !IsReservedOrderExtraKey("PRICE") || !IsReservedOrderExtraKey("id") {
		t.Fatal("expected reserved")
	}
}

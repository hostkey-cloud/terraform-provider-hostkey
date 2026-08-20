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

func TestEQOrderInstance_IPv4AmountIsAdditionalNotTotal(t *testing.T) {
	// Regression: InvAPI bills ipv4_amount as additional addresses beyond the
	// default one. Schema total 1 must omit the param; totals >1 send (n-1).
	cases := []struct {
		name      string
		requested int
		wantParam string
		wantSet   bool
	}{
		{"unset", 0, "", false},
		{"default_single_address", 1, "", false},
		{"one_extra_address", 2, "1", true},
		{"three_extra_addresses", 4, "3", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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
				Preset:       "vm.pico",
				LocationName: "NL",
				RootPass:     "Abcdef1%",
				IPv4Amount:   tc.requested,
			})
			if err != nil {
				t.Fatal(err)
			}

			_, isSet := captured["ipv4_amount"]
			if isSet != tc.wantSet {
				t.Fatalf("ipv4_amount set=%v, want %v (raw=%q)", isSet, tc.wantSet, captured.Get("ipv4_amount"))
			}
			if isSet && captured.Get("ipv4_amount") != tc.wantParam {
				t.Fatalf("ipv4_amount=%q, want %q", captured.Get("ipv4_amount"), tc.wantParam)
			}
		})
	}
}

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

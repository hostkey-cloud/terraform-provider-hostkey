package invapi

import (
	"strings"
	"testing"
)

func TestValidateBaseURL(t *testing.T) {
	for _, u := range []string{
		"https://invapi.hostkey.com/",
		"https://invapi.hostkey.ru",
		"http://127.0.0.1:8080/",
		"http://localhost/invapi/",
	} {
		if err := ValidateBaseURL(u); err != nil {
			t.Fatalf("%q: %v", u, err)
		}
	}
	for _, bad := range []string{
		"",
		"ftp://x",
		"not-a-url",
		"https://",
		"http://invapi.hostkey.com/",
		"http://evil.example/",
	} {
		if err := ValidateBaseURL(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestAllowedInvAPIRewrite(t *testing.T) {
	cur := "https://invapi.hostkey.com/"
	ok, err := CanonicalInvAPIBaseURL("invapi.hostkey.ru")
	if err != nil {
		t.Fatal(err)
	}
	if err := allowedInvAPIRewrite(cur, ok); err != nil {
		t.Fatalf("hostkey.ru rewrite: %v", err)
	}
	if err := allowedInvAPIRewrite(cur, "https://evil.example/"); err == nil {
		t.Fatal("expected reject attacker host")
	}
	if err := allowedInvAPIRewrite(cur, "http://invapi.hostkey.com/"); err == nil {
		t.Fatal("expected reject TLS downgrade")
	}
}

func TestIsHostkeyAPIHost(t *testing.T) {
	if !isHostkeyAPIHost("invapi.hostkey.com") || !isHostkeyAPIHost("INVAPI.HOSTKEY.RU") {
		t.Fatal("expected hostkey hosts")
	}
	if isHostkeyAPIHost("hostkey.com.evil.example") || isHostkeyAPIHost("example.com") {
		t.Fatal("expected reject lookalike")
	}
}

func TestRedactSecrets(t *testing.T) {
	got := redactSecrets(`{"token":"abc123","key":"secret"}`)
	if strings.Contains(got, "abc123") || strings.Contains(got, `"key":"secret"`) {
		t.Fatalf("not redacted: %s", got)
	}
	form := redactSecrets("token=sess&root_pass=Abcdef1%")
	if strings.Contains(form, "sess") || strings.Contains(form, "Abcdef1%") {
		t.Fatalf("form not redacted: %s", form)
	}
}

package provider

import "testing"

func TestValidateRootPass(t *testing.T) {
	if err := validateRootPass("87GxtAkn5R+"); err != nil {
		t.Fatalf("valid pass rejected: %v", err)
	}
	if err := validateRootPass("short1A+"); err != nil {
		t.Fatalf("8-char valid rejected: %v", err)
	}
	if err := validateRootPass("87GxtAkn5R"); err == nil {
		t.Fatal("expected missing special char")
	}
	if err := validateRootPass("+BadStart1"); err == nil {
		t.Fatal("expected leading special rejected")
	}
	if err := validateRootPass("Bad@Pass1+"); err == nil {
		t.Fatal("expected @ rejected")
	}
}

func TestPendingID(t *testing.T) {
	id := pendingID(995884)
	if id != "pending:995884" {
		t.Fatalf("got %s", id)
	}
	n, ok := parsePendingInvoice(id)
	if !ok || n != 995884 {
		t.Fatalf("parse failed: %d %v", n, ok)
	}
}

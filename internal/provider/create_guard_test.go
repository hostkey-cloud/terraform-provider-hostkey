package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hostkey-cloud/terraform-provider-hostkey/internal/invapi"
)

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
	const invoice = 123456
	id := pendingID(invoice)
	if id != "pending:123456" {
		t.Fatalf("got %s", id)
	}
	n, ok := parsePendingInvoice(id)
	if !ok || n != invoice {
		t.Fatalf("parse failed: %d %v", n, ok)
	}
}

func TestAcceptNewServerID(t *testing.T) {
	known := map[int]struct{}{100: {}, 200: {}}
	if err := acceptNewServerID(300, known); err != nil {
		t.Fatalf("new id: %v", err)
	}
	if err := acceptNewServerID(100, known); err == nil {
		t.Fatal("expected error for pre-existing id")
	}
	if err := acceptNewServerID(0, known); err == nil {
		t.Fatal("expected error for zero id")
	}
}

func TestSnapshotKnownIDs(t *testing.T) {
	_, err := snapshotKnownIDs(nil)
	if err == nil {
		t.Fatal("nil list should error")
	}
	list := &invapi.ServerListResponse{Servers: []byte(`[10,20]`)}
	known, err := snapshotKnownIDs(list)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := known[10]; !ok {
		t.Fatal("missing 10")
	}
	empty := &invapi.ServerListResponse{Servers: []byte(`[]`)}
	known, err = snapshotKnownIDs(empty)
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 0 {
		t.Fatalf("empty list: %#v", known)
	}
}

func TestPendingInvoiceFromState(t *testing.T) {
	n, ok := pendingInvoiceFromState(serverModel{ID: types.StringValue("pending:603548")})
	if !ok || n != 603548 {
		t.Fatalf("id field: %d %v", n, ok)
	}
	n, ok = pendingInvoiceFromState(serverModel{
		ID:      types.StringValue("pending:x"),
		Invoice: types.Int64Value(99),
	})
	if !ok || n != 99 {
		t.Fatalf("invoice field: %d %v", n, ok)
	}
}

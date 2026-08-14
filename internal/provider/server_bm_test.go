package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildOrderRequestBareMetal(t *testing.T) {
	plan := serverModel{
		LocationName: types.StringValue("NL"),
		RootPass:     types.StringValue("Abcdef1%"),
		PresetName:   types.StringValue("bm.v2-max"),
		DiskMirror:   types.StringValue("RAID1"),
		NoLVM:        types.BoolValue(true),
		IPv6Block:    types.BoolValue(true),
		IPv4Amount:   types.Int64Value(1),
	}
	req := buildOrderRequest(plan)
	if req.DiskMirror != "raid1" {
		t.Fatalf("disk_mirror: got %q want raid1", req.DiskMirror)
	}
	if req.NoLVM == nil || !*req.NoLVM {
		t.Fatal("expected no_lvm=true")
	}
	if req.IPv6Block == nil || !*req.IPv6Block {
		t.Fatal("expected ipv6_block=true")
	}
	if req.IPv4Amount != 1 {
		t.Fatalf("ipv4_amount: got %d", req.IPv4Amount)
	}
}

func TestNeedsReinstallDiskOptions(t *testing.T) {
	base := serverModel{
		OSName:     types.StringValue("Ubuntu 22.04"),
		OSID:       types.Int64Value(187),
		RootPass:   types.StringValue("Abcdef1%"),
		DiskMirror: types.StringValue("hba"),
		NoLVM:      types.BoolValue(false),
	}
	raid := base
	raid.DiskMirror = types.StringValue("raid1")
	if !needsReinstall(raid, base) {
		t.Fatal("expected reinstall on disk_mirror change")
	}
	lvm := base
	lvm.NoLVM = types.BoolValue(true)
	if !needsReinstall(lvm, base) {
		t.Fatal("expected reinstall on no_lvm change")
	}
}

func TestValidateBareMetalOrderOptions(t *testing.T) {
	diags := validateBareMetalOrderOptions(serverModel{
		PresetName:   types.StringValue("bm.v2-promo"),
		LocationName: types.StringValue("DE"),
		IPv6Block:    types.BoolValue(true),
	})
	if !diags.HasError() {
		t.Fatal("expected error for ipv6_block outside NL/US")
	}

	diags = validateBareMetalOrderOptions(serverModel{
		PresetName:   types.StringValue("bm.v2-promo"),
		LocationName: types.StringValue("NL"),
		IPv6Block:    types.BoolValue(true),
	})
	if diags.HasError() {
		t.Fatalf("NL dedicated ipv6 location check should pass: %v", diags.Errors())
	}
}

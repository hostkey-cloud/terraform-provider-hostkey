package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNeedsReinstall(t *testing.T) {
	base := serverModel{
		OSName:   types.StringValue("Ubuntu 22.04"),
		OSID:     types.Int64Value(187),
		RootPass: types.StringValue("Abcdef1%"),
	}

	same := base
	if needsReinstall(same, base) {
		t.Fatal("expected no reinstall when unchanged")
	}

	osChange := base
	osChange.OSName = types.StringValue("Ubuntu 24.04")
	osChange.OSID = types.Int64Value(237)
	if !needsReinstall(osChange, base) {
		t.Fatal("expected reinstall on OS change")
	}

	passChange := base
	passChange.RootPass = types.StringValue("Abcdef2%")
	if !needsReinstall(passChange, base) {
		t.Fatal("expected reinstall on root_pass change")
	}

	trigger := base
	trigger.ReinstallTrigger = types.StringValue("wipe-1")
	stateNoTrig := base
	if !needsReinstall(trigger, stateNoTrig) {
		t.Fatal("expected reinstall on reinstall_trigger set")
	}

	presetOnly := base
	presetOnly.PresetName = types.StringValue("vm.mini")
	statePreset := base
	statePreset.PresetName = types.StringValue("vm.pico")
	if needsReinstall(presetOnly, statePreset) {
		t.Fatal("preset change is replace, not reinstall")
	}

	imported := base
	importedState := serverModel{
		OSName: types.StringNull(),
		OSID:   types.Int64Null(),
	}
	if needsReinstall(imported, importedState) {
		t.Fatal("imported server with null install fields must not reinstall on first apply")
	}

	importedPass := base
	importedPassState := serverModel{RootPass: types.StringNull()}
	if needsReinstall(importedPass, importedPassState) {
		t.Fatal("root_pass on imported server must not reinstall when state had no password")
	}
}

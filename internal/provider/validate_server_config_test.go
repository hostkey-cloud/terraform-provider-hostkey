package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateServerConfigPowerOffHardRequiresOff(t *testing.T) {
	config := serverModel{
		PowerState:   types.StringValue("on"),
		PowerOffHard: types.BoolValue(true),
	}
	diags := validateServerConfig(config)
	if !diags.HasError() {
		t.Fatal("expected error for power_off_hard with power_state=on")
	}
}

func TestValidateServerConfigPowerOffHardWithOffOK(t *testing.T) {
	config := serverModel{
		PowerState:   types.StringValue("off"),
		PowerOffHard: types.BoolValue(true),
	}
	diags := validateServerConfig(config)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
}

func TestValidateServerConfigOwnOSWarnsOnOSName(t *testing.T) {
	config := serverModel{
		OwnOS:  types.BoolValue(true),
		OSName: types.StringValue("Ubuntu 22.04"),
	}
	diags := validateServerConfig(config)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if len(diags.Warnings()) == 0 {
		t.Fatal("expected a warning for own_os with os_name set")
	}
}

func TestValidateServerConfigIsStaticNoNetwork(t *testing.T) {
	// validateServerConfig must not require a client/context to run its
	// checks -- this is the whole point of moving it out of ModifyPlan and
	// into ValidateConfig. This test is a regression guard on the
	// function signature itself: config-only in, diagnostics out.
	config := serverModel{
		LocationName: types.StringValue("DE"),
		IPv6Block:    types.BoolValue(true),
	}
	diags := validateServerConfig(config)
	if !diags.HasError() {
		t.Fatal("expected ipv6_block/location error to surface from validateServerConfig alone")
	}
}

func TestValidateServerTags(t *testing.T) {
	longVal := types.StringValue(string(make([]byte, maxTagValueLen+1)))
	diags := validateServerTags(types.MapValueMust(types.StringType, map[string]attr.Value{
		"k": longVal,
	}))
	if !diags.HasError() {
		t.Fatal("expected tag value length error")
	}
}

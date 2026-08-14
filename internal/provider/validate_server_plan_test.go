package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateServerPlanCreateRequiresPresetOSAndTraffic(t *testing.T) {
	plan := serverModel{
		LocationName: types.StringValue("NL"),
		RootPass:     types.StringValue("Abcdef1%"),
	}
	diags := validateServerPlan(context.Background(), plan, serverModel{}, true)
	if !diags.HasError() {
		t.Fatal("expected errors for missing preset/os/traffic")
	}
}

func TestValidateServerPlanCreateOK(t *testing.T) {
	plan := serverModel{
		LocationName:    types.StringValue("NL"),
		RootPass:        types.StringValue("Abcdef1%"),
		PresetName:      types.StringValue("vm.pico"),
		OSName:          types.StringValue("Ubuntu 22.04"),
		TrafficPlanName: types.StringValue("3 TB / 1 Gbps VM"),
	}
	diags := validateServerPlan(context.Background(), plan, serverModel{}, true)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
}

func TestValidateServerPlanPowerOffHardRequiresOff(t *testing.T) {
	plan := serverModel{
		PowerState:   types.StringValue("on"),
		PowerOffHard: types.BoolValue(true),
	}
	diags := validateServerPlan(context.Background(), plan, serverModel{ID: types.StringValue("100")}, false)
	if !diags.HasError() {
		t.Fatal("expected error for power_off_hard with power_state=on")
	}
}

func TestValidateServerPlanReservedExtraOrderParams(t *testing.T) {
	for _, key := range []string{"token", "action", "id", "price", "preset", "ipv6", "disk_mirror", "uefi"} {
		plan := serverModel{
			ExtraOrderParams: types.MapValueMust(types.StringType, map[string]attr.Value{
				key: types.StringValue("x"),
			}),
		}
		diags := validateServerPlan(context.Background(), plan, serverModel{ID: types.StringValue("100")}, false)
		if !diags.HasError() {
			t.Fatalf("expected error for reserved extra_order_params key %q", key)
		}
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

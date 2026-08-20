package provider

import (
	"context"
	"testing"

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

func TestValidateServerPlanNotCreateSkipsRequiredFieldChecks(t *testing.T) {
	// On update (isCreate=false), missing preset/os/traffic must not error --
	// those fields are already fixed by the prior create/order and this
	// check only guards the initial order.
	plan := serverModel{
		LocationName: types.StringValue("NL"),
		RootPass:     types.StringValue("Abcdef1%"),
	}
	diags := validateServerPlan(context.Background(), plan, serverModel{ID: types.StringValue("100")}, false)
	if diags.HasError() {
		t.Fatalf("unexpected errors on update: %v", diags.Errors())
	}
}

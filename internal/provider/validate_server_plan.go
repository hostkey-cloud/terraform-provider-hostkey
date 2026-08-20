package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// validateServerPlan checks the create-only "you must specify a preset/OS/
// traffic plan" requirements. This intentionally stays in ModifyPlan (rather
// than moving to ValidateConfig alongside the rest of the static checks in
// validate_server_config.go) because it is gated on isCreate, which requires
// comparing against prior state -- state is not available to ValidateConfig.
func validateServerPlan(_ context.Context, plan, _ serverModel, isCreate bool) diag.Diagnostics {
	var diags diag.Diagnostics

	if !isCreate {
		return diags
	}

	hasPreset := (!plan.PresetID.IsNull() && plan.PresetID.ValueInt64() > 0) ||
		(!plan.PresetName.IsNull() && strings.TrimSpace(plan.PresetName.ValueString()) != "")
	if !hasPreset {
		diags.AddAttributeError(
			path.Root("preset_name"),
			"Missing preset",
			"Set preset_name or preset_id when creating a new server.",
		)
	}

	ownOS := !plan.OwnOS.IsNull() && plan.OwnOS.ValueBool()
	hasOS := (!plan.OSID.IsNull() && plan.OSID.ValueInt64() > 0) ||
		(!plan.OSName.IsNull() && strings.TrimSpace(plan.OSName.ValueString()) != "")
	hasTemplate := !plan.OSTemplate.IsNull() && strings.TrimSpace(plan.OSTemplate.ValueString()) != ""
	if !ownOS && !hasOS && !hasTemplate {
		diags.AddAttributeError(
			path.Root("os_name"),
			"Missing OS",
			"Set os_name or os_id when creating a server, unless own_os=true or os_template is set.",
		)
	}

	hasTraffic := (!plan.TrafficPlanID.IsNull() && plan.TrafficPlanID.ValueInt64() > 0) ||
		(!plan.TrafficPlanName.IsNull() && strings.TrimSpace(plan.TrafficPlanName.ValueString()) != "")
	if !hasTraffic {
		diags.AddAttributeError(
			path.Root("traffic_plan_name"),
			"Missing traffic plan",
			"Set traffic_plan_name or traffic_plan_id when creating a server.",
		)
	}

	return diags
}

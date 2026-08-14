package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateServerPlan(_ context.Context, plan, _ serverModel, isCreate bool) diag.Diagnostics {
	var diags diag.Diagnostics

	if isCreate {
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
	}

	if !plan.OwnOS.IsNull() && plan.OwnOS.ValueBool() {
		if !plan.OSName.IsNull() && plan.OSName.ValueString() != "" {
			diags.AddAttributeWarning(
				path.Root("own_os"),
				"own_os ignores OS selection",
				"own_os=true skips OS install; os_name/os_id are ignored by InvAPI.",
			)
		}
	}

	if !plan.PowerOffHard.IsNull() && plan.PowerOffHard.ValueBool() {
		ps := strings.ToLower(strings.TrimSpace(plan.PowerState.ValueString()))
		if ps != "" && ps != "off" {
			diags.AddAttributeError(
				path.Root("power_off_hard"),
				"power_off_hard requires power_state=off",
				fmt.Sprintf("power_off_hard is only used when power_state is \"off\"; got %q.", plan.PowerState.ValueString()),
			)
		}
	}

	if !plan.ExtraOrderParams.IsNull() && !plan.ExtraOrderParams.IsUnknown() {
		raw := map[string]types.String{}
		_ = plan.ExtraOrderParams.ElementsAs(context.Background(), &raw, false)
		for k := range raw {
			diags.AddAttributeError(
				path.Root("extra_order_params"),
				"extra_order_params is closed",
				fmt.Sprintf("key %q is not allowed. All eq/order_instance fields are typed attributes; extra_order_params is not forwarded.", k),
			)
		}
	}

	diags.Append(validateBareMetalOrderOptions(plan)...)
	diags.Append(validateServerTags(plan.Tags)...)

	return diags
}

func validateServerTags(tags types.Map) diag.Diagnostics {
	var diags diag.Diagnostics
	if tags.IsNull() || tags.IsUnknown() {
		return diags
	}
	raw := map[string]types.String{}
	_ = tags.ElementsAs(context.Background(), &raw, false)
	for k, v := range raw {
		if strings.TrimSpace(k) == "" {
			diags.AddAttributeError(path.Root("tags"), "Invalid tag key", "tag keys must not be empty.")
			continue
		}
		if len(k) > maxTagKeyLen {
			diags.AddAttributeError(path.Root("tags"), "Invalid tag key",
				fmt.Sprintf("tag key %q exceeds %d characters.", k, maxTagKeyLen))
		}
		if !v.IsNull() && len(v.ValueString()) > maxTagValueLen {
			diags.AddAttributeError(path.Root("tags"), "Invalid tag value",
				fmt.Sprintf("tag %q value exceeds %d characters.", k, maxTagValueLen))
		}
	}
	return diags
}

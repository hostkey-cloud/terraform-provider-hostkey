package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// validateServerConfig holds the purely static, zero-network hostkey_server
// cross-field checks. Unlike validateServerPlan (create-only, needs prior
// state) and verifyOrderCatalog (network-backed, needs the provider client),
// every check here only inspects the configuration as written, so it runs
// from ValidateConfig -- visible to a bare `terraform validate` with no
// provider credentials, and no longer coupled to the API-backed catalog
// checks that used to share the same ModifyPlan hook.
func validateServerConfig(config serverModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if !config.OwnOS.IsNull() && config.OwnOS.ValueBool() {
		if !config.OSName.IsNull() && config.OSName.ValueString() != "" {
			diags.AddAttributeWarning(
				path.Root("own_os"),
				"own_os ignores OS selection",
				"own_os=true skips OS install; os_name/os_id are ignored by InvAPI.",
			)
		}
	}

	if !config.OSTemplate.IsNull() && strings.TrimSpace(config.OSTemplate.ValueString()) != "" {
		diags.AddAttributeWarning(
			path.Root("os_template"),
			"Custom OS template is opaque and unvalidated",
			"os_template is forwarded to InvAPI's eq/order_instance as-is; Hostkey API docs do not publish a discoverable list of valid os_template values, so this provider cannot validate it beyond a length cap. os_name/os_id catalog checks do not apply when os_template is set.",
		)
	}

	if !config.DeployOptions.IsNull() && strings.TrimSpace(config.DeployOptions.ValueString()) != "" {
		diags.AddAttributeWarning(
			path.Root("deploy_options"),
			"Opaque deploy_options is unvalidated",
			"deploy_options is forwarded to InvAPI's eq/order_instance as-is; Hostkey API docs do not publish its accepted keys/format for this generic string, so this provider cannot validate its contents beyond a length cap. Invalid values fail only at order/reinstall time.",
		)
	}

	if !config.IPv4Amount.IsNull() && !config.IPv4Amount.IsUnknown() && config.IPv4Amount.ValueInt64() > 1 {
		diags.AddAttributeWarning(
			path.Root("ipv4_amount"),
			"Extra IPv4 addresses may be billed and are not quota-checked",
			fmt.Sprintf("ipv4_amount=%d requests additional IPv4 addresses beyond the default single address. Extra IPv4s may incur recurring charges depending on the location/account. The 1-%d bound is a conservative static guess, not a documented InvAPI quota.", config.IPv4Amount.ValueInt64(), maxIPv4Amount),
		)
	}

	if !config.VLAN.IsNull() && !config.VLAN.IsUnknown() {
		diags.AddAttributeWarning(
			path.Root("vlan"),
			"vlan is not validated against your account",
			fmt.Sprintf("vlan=%d is forwarded to InvAPI as-is. No account VLAN list/quota endpoint is published in Hostkey API docs, so this value is only range-checked (>=1), not verified to belong to you. An invalid vlan id fails only at order/reinstall time.", config.VLAN.ValueInt64()),
		)
	}
	if !config.PrivateVLAN.IsNull() && !config.PrivateVLAN.IsUnknown() {
		diags.AddAttributeWarning(
			path.Root("private_vlan"),
			"private_vlan is not validated against your account",
			fmt.Sprintf("private_vlan=%d is forwarded to InvAPI as-is. Per Hostkey API docs, private_vlan/private_ip must be pre-reserved via ipv4/reserve before use; this provider cannot verify that reservation exists. An unreserved/invalid private_vlan fails only at order time.", config.PrivateVLAN.ValueInt64()),
		)
	}

	if !config.PowerOffHard.IsNull() && config.PowerOffHard.ValueBool() {
		ps := strings.ToLower(strings.TrimSpace(config.PowerState.ValueString()))
		if ps != "" && ps != "off" {
			diags.AddAttributeError(
				path.Root("power_off_hard"),
				"power_off_hard requires power_state=off",
				fmt.Sprintf("power_off_hard is only used when power_state is \"off\"; got %q.", config.PowerState.ValueString()),
			)
		}
	}

	diags.Append(validateBareMetalOrderOptions(config)...)
	diags.Append(validateServerTags(config.Tags)...)

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

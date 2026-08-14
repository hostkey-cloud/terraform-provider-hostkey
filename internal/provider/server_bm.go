package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func diskMirrorValidator() validator.String {
	return oneOfStrings("disk_mirror", "hba", "raid0", "raid1", "raid10")
}

func validateBareMetalOrderOptions(plan serverModel) diag.Diagnostics {
	var diags diag.Diagnostics

	hasIPv6 := !plan.IPv6Block.IsNull() && !plan.IPv6Block.IsUnknown() && plan.IPv6Block.ValueBool()
	if hasIPv6 && !plan.LocationName.IsNull() && !plan.LocationName.IsUnknown() {
		loc := strings.ToUpper(strings.TrimSpace(plan.LocationName.ValueString()))
		if loc != "" && loc != "NL" && loc != "US" {
			diags.AddAttributeError(
				path.Root("ipv6_block"),
				"IPv6 /64 not available in this location",
				fmt.Sprintf("Hostkey documents IPv6 /64 for dedicated servers only in NL and US; got location_name=%q.", plan.LocationName.ValueString()),
			)
		}
	}

	return diags
}

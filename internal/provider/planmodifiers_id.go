package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// requiresReplaceOnIDChange replaces only when a known id in state changes to a different known id.
// Null/unknown → first value (import or name resolve) does not force replace.
func requiresReplaceOnIDChange() planmodifier.Int64 {
	return int64planmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.Int64Request, resp *int64planmodifier.RequiresReplaceIfFuncResponse) {
			if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
				resp.RequiresReplace = false
				return
			}
			if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
				resp.RequiresReplace = false
				return
			}
			resp.RequiresReplace = !req.PlanValue.Equal(req.StateValue)
		},
		"Replace when catalog id changes",
		"Replace when catalog id changes",
	)
}

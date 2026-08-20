package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// requiresReplaceOnIDChange replaces only when a known id in state changes to
// a different known id. Null/unknown state (e.g. right after import, or on
// first create before the id resolves) never forces replace by itself.
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

// requiresReplaceOnKnown{String,Int64,Bool}Change: force replace only when BOTH
// prior state and planned values are known AND differ. A null/unknown prior
// state (typical right after import for attributes InvAPI cannot round-trip)
// never forces replace by itself — otherwise the first plan after import
// destroys a live server when the user fills matching HCL.
func requiresReplaceOnKnownStringChange() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
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
		"Replace when value changes (a null/unknown prior state, e.g. right after import, never forces replace by itself)",
		"Replace when value changes (a null/unknown prior state, e.g. right after import, never forces replace by itself)",
	)
}

func requiresReplaceOnKnownInt64Change() planmodifier.Int64 {
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
		"Replace when value changes (a null/unknown prior state, e.g. right after import, never forces replace by itself)",
		"Replace when value changes (a null/unknown prior state, e.g. right after import, never forces replace by itself)",
	)
}

func requiresReplaceOnKnownBoolChange() planmodifier.Bool {
	return boolplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.BoolRequest, resp *boolplanmodifier.RequiresReplaceIfFuncResponse) {
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
		"Replace when value changes (a null/unknown prior state, e.g. right after import, never forces replace by itself)",
		"Replace when value changes (a null/unknown prior state, e.g. right after import, never forces replace by itself)",
	)
}

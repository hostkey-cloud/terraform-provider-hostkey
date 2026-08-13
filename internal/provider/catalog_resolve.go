package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hostkey-cloud/terraform-provider-hostkey/internal/invapi"
)

type namedID struct {
	ID   int
	Name string
}

func matchNamedID(query string, items []namedID) (int, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, fmt.Errorf("empty name")
	}

	var exact []namedID
	for _, it := range items {
		if strings.EqualFold(strings.TrimSpace(it.Name), q) {
			exact = append(exact, it)
		}
	}
	if len(exact) == 1 {
		return exact[0].ID, nil
	}
	if len(exact) > 1 {
		return 0, fmt.Errorf("name %q matches multiple entries: %s", q, joinNames(exact))
	}

	var partial []namedID
	for _, it := range items {
		if containsFold(it.Name, q) {
			partial = append(partial, it)
		}
	}
	if len(partial) == 1 {
		return partial[0].ID, nil
	}
	if len(partial) > 1 {
		return 0, fmt.Errorf("name %q is ambiguous (%d matches): %s — use a more specific name or the numeric id", q, len(partial), joinNames(partial))
	}
	return 0, fmt.Errorf("name %q not found", q)
}

func joinNames(items []namedID) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("%q (id=%d)", it.Name, it.ID))
	}
	return strings.Join(parts, ", ")
}

func resolvePresetID(ctx context.Context, client *invapi.Client, location, name string) (int, error) {
	list, err := client.PresetsList(ctx, invapi.PresetsListFilter{Location: location})
	if err != nil {
		return 0, err
	}
	items := make([]namedID, 0, len(list.Presets))
	for _, p := range list.Presets {
		items = append(items, namedID{ID: p.ID, Name: p.Name})
	}
	return matchNamedID(name, items)
}

func resolveOSID(ctx context.Context, client *invapi.Client, location string, presetID int, name string) (int, error) {
	list, err := client.OSList(ctx, invapi.OSListFilter{
		Location:   location,
		InstanceID: presetID,
	})
	if err != nil {
		return 0, err
	}
	items := make([]namedID, 0, len(list.OSList))
	for _, o := range list.OSList {
		items = append(items, namedID{ID: o.ID, Name: o.Name})
	}
	return matchNamedID(name, items)
}

func resolveSoftID(ctx context.Context, client *invapi.Client, location string, presetID int, name string) (int, error) {
	list, err := client.SoftwareList(ctx, invapi.SoftwareListFilter{
		Location:   location,
		InstanceID: presetID,
	})
	if err != nil {
		return 0, err
	}
	items := make([]namedID, 0, len(list.Software))
	for _, s := range list.Software {
		items = append(items, namedID{ID: s.ID, Name: s.Name})
	}
	return matchNamedID(name, items)
}

func resolveTrafficPlanID(ctx context.Context, client *invapi.Client, location string, presetID int, name string) (int, error) {
	if location == "" {
		return 0, fmt.Errorf("location_name is required to resolve traffic_plan_name")
	}
	list, err := client.TrafficPlansList(ctx, invapi.TrafficPlansListFilter{
		Location:   location,
		InstanceID: presetID,
	})
	if err != nil {
		return 0, err
	}
	items := make([]namedID, 0, len(list.TrafficPlans))
	for _, p := range list.TrafficPlans {
		items = append(items, namedID{ID: p.ID, Name: p.Name})
	}
	return matchNamedID(name, items)
}

// resolveOrderIDs fills numeric IDs from human-readable names on the plan.
func (r *serverResource) resolveOrderIDs(ctx context.Context, plan *serverModel) error {
	if r.client == nil {
		return fmt.Errorf("provider not configured")
	}
	location := plan.LocationName.ValueString()
	var errs []string

	presetID := 0
	if !plan.PresetID.IsNull() && !plan.PresetID.IsUnknown() {
		presetID = int(plan.PresetID.ValueInt64())
	}
	if presetID == 0 && !plan.PresetName.IsNull() && plan.PresetName.ValueString() != "" {
		id, err := resolvePresetID(ctx, r.client, location, plan.PresetName.ValueString())
		if err != nil {
			errs = append(errs, fmt.Sprintf("preset_name: %v", err))
		} else {
			presetID = id
			plan.PresetID = typesInt64(id)
		}
	}

	if (plan.OSID.IsNull() || plan.OSID.IsUnknown() || plan.OSID.ValueInt64() == 0) &&
		!plan.OSName.IsNull() && plan.OSName.ValueString() != "" {
		id, err := resolveOSID(ctx, r.client, location, presetID, plan.OSName.ValueString())
		if err != nil {
			errs = append(errs, fmt.Sprintf("os_name: %v", err))
		} else {
			plan.OSID = typesInt64(id)
		}
	}

	if (plan.SoftID.IsNull() || plan.SoftID.IsUnknown() || plan.SoftID.ValueInt64() == 0) &&
		!plan.SoftName.IsNull() && plan.SoftName.ValueString() != "" {
		id, err := resolveSoftID(ctx, r.client, location, presetID, plan.SoftName.ValueString())
		if err != nil {
			errs = append(errs, fmt.Sprintf("soft_name: %v", err))
		} else {
			plan.SoftID = typesInt64(id)
		}
	}

	if (plan.TrafficPlanID.IsNull() || plan.TrafficPlanID.IsUnknown() || plan.TrafficPlanID.ValueInt64() == 0) &&
		!plan.TrafficPlanName.IsNull() && plan.TrafficPlanName.ValueString() != "" {
		id, err := resolveTrafficPlanID(ctx, r.client, location, presetID, plan.TrafficPlanName.ValueString())
		if err != nil {
			errs = append(errs, fmt.Sprintf("traffic_plan_name: %v", err))
		} else {
			plan.TrafficPlanID = typesInt64(id)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func typesInt64(id int) types.Int64 {
	return types.Int64Value(int64(id))
}

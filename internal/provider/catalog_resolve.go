package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hostkey-cloud/terraform-provider-hostkey/internal/invapi"
)

type namedID struct {
	ID   int
	Name string
}

type trafficNamedID struct {
	ID    int
	Name  string
	Price float64
}

var (
	trafficFreeSuffix = regexp.MustCompile(`(?i)\s*-\s*FREE\s*$`)
	trafficPriceParen = regexp.MustCompile(`(?i)\s*\((\d+(?:\.\d+)?)\s*P\)\s*$`)
)

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
	return 0, fmt.Errorf("name %q not found (exact catalog match required)", q)
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
	actives := make([]int, 0, len(list.OSList))
	for _, o := range list.OSList {
		items = append(items, namedID{ID: o.ID, Name: o.Name})
		actives = append(actives, o.Active)
	}
	return matchNamedID(name, filterActiveNamedIDs(items, actives))
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
	actives := make([]int, 0, len(list.Software))
	for _, s := range list.Software {
		items = append(items, namedID{ID: s.ID, Name: s.Name})
		actives = append(actives, s.Active)
	}
	return matchNamedID(name, filterActiveNamedIDs(items, actives))
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
	items := make([]trafficNamedID, 0, len(list.TrafficPlans))
	for _, p := range list.TrafficPlans {
		if p.Active == 0 {
			continue
		}
		items = append(items, trafficNamedID{ID: p.ID, Name: p.Name, Price: p.Price})
	}
	if len(items) == 0 {
		return 0, fmt.Errorf("no active traffic plans for preset in location %s", location)
	}
	return matchTrafficPlan(name, items)
}

// matchTrafficPlan resolves InvAPI traffic plan names.
// Dedicated catalogs often expose duplicate names distinguished only by price
// (panel labels like "1Gbps 50TB - FREE" / "1Gbps unmetered (10000 P)").
func matchTrafficPlan(query string, items []trafficNamedID) (int, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, fmt.Errorf("empty name")
	}

	var wantPrice *float64
	base := q
	if trafficFreeSuffix.MatchString(base) {
		base = strings.TrimSpace(trafficFreeSuffix.ReplaceAllString(base, ""))
		z := 0.0
		wantPrice = &z
	} else if m := trafficPriceParen.FindStringSubmatch(base); len(m) == 2 {
		base = strings.TrimSpace(trafficPriceParen.ReplaceAllString(base, ""))
		p, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid price hint in %q: %w", query, err)
		}
		wantPrice = &p
	}

	exact := filterTrafficByName(base, items, true)
	if len(exact) == 0 {
		exact = filterTrafficByName(base, items, false)
	}
	if len(exact) == 0 {
		return 0, fmt.Errorf("name %q not found", query)
	}
	if wantPrice != nil {
		var priced []trafficNamedID
		for _, it := range exact {
			if it.Price == *wantPrice {
				priced = append(priced, it)
			}
		}
		if len(priced) == 1 {
			return priced[0].ID, nil
		}
		if len(priced) == 0 {
			return 0, fmt.Errorf("name %q: no plan with price %g; candidates: %s", query, *wantPrice, joinTrafficNames(exact))
		}
		exact = priced
	}
	if len(exact) == 1 {
		return exact[0].ID, nil
	}
	return 0, fmt.Errorf("name %q is ambiguous (%d matches): %s — use traffic_plan_id or a panel-style name with price hint (e.g. \"… - FREE\", \"… (10000 P)\")", query, len(exact), joinTrafficNames(exact))
}

func filterTrafficByName(query string, items []trafficNamedID, exact bool) []trafficNamedID {
	var out []trafficNamedID
	for _, it := range items {
		name := strings.TrimSpace(it.Name)
		if exact {
			if strings.EqualFold(name, query) {
				out = append(out, it)
			}
			continue
		}
		// Price-hint path already stripped the suffix; still require exact base name.
		if strings.EqualFold(name, query) {
			out = append(out, it)
		}
	}
	return out
}

func filterActiveNamedIDs(items []namedID, active []int) []namedID {
	if len(items) != len(active) {
		return items
	}
	out := make([]namedID, 0, len(items))
	for i, it := range items {
		if active[i] != 0 {
			out = append(out, it)
		}
	}
	return out
}

func joinTrafficNames(items []trafficNamedID) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("%q (id=%d, price=%g)", it.Name, it.ID, it.Price))
	}
	return strings.Join(parts, ", ")
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
	if !plan.PresetName.IsNull() && !plan.PresetName.IsUnknown() && strings.TrimSpace(plan.PresetName.ValueString()) != "" {
		id, err := resolvePresetID(ctx, r.client, location, plan.PresetName.ValueString())
		if err != nil {
			errs = append(errs, fmt.Sprintf("preset_name: %v", err))
		} else {
			presetID = id
			plan.PresetID = typesInt64(id)
		}
	}

	// When *_name is set, always sync *_id from the catalog. Computed ids copied from
	// state (UseStateForUnknown) would otherwise stay stale after a name-only change.
	if !plan.OSName.IsNull() && !plan.OSName.IsUnknown() && strings.TrimSpace(plan.OSName.ValueString()) != "" {
		id, err := resolveOSID(ctx, r.client, location, presetID, plan.OSName.ValueString())
		if err != nil {
			errs = append(errs, fmt.Sprintf("os_name: %v", err))
		} else {
			plan.OSID = typesInt64(id)
		}
	}

	if !plan.SoftName.IsNull() && !plan.SoftName.IsUnknown() && strings.TrimSpace(plan.SoftName.ValueString()) != "" {
		id, err := resolveSoftID(ctx, r.client, location, presetID, plan.SoftName.ValueString())
		if err != nil {
			errs = append(errs, fmt.Sprintf("soft_name: %v", err))
		} else {
			plan.SoftID = typesInt64(id)
		}
	}

	if !plan.TrafficPlanName.IsNull() && !plan.TrafficPlanName.IsUnknown() && strings.TrimSpace(plan.TrafficPlanName.ValueString()) != "" {
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

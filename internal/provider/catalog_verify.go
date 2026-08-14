package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hostkey-cloud/terraform-provider-hostkey/internal/invapi"
)

func catalogHasID(ids []int, want int) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func (r *serverResource) verifyOrderCatalog(ctx context.Context, plan *serverModel) error {
	if r.client == nil {
		return nil
	}
	if plan.LocationName.IsNull() || plan.LocationName.IsUnknown() {
		return nil
	}
	location := plan.LocationName.ValueString()
	if location == "" {
		return nil
	}

	presetID := 0
	if !plan.PresetID.IsNull() && !plan.PresetID.IsUnknown() {
		presetID = int(plan.PresetID.ValueInt64())
	}

	list, err := r.client.PresetsList(ctx, invapi.PresetsListFilter{Location: location})
	if err != nil {
		return fmt.Errorf("presets/list: %w", err)
	}
	var presetName string
	var presetIDs []int
	for _, p := range list.Presets {
		presetIDs = append(presetIDs, p.ID)
		if presetID > 0 && p.ID == presetID {
			presetName = p.Name
		}
	}

	if presetID > 0 && !catalogHasID(presetIDs, presetID) {
		return fmt.Errorf("preset_id %d is not in presets/list for location %s", presetID, location)
	}

	if !plan.PresetName.IsNull() && !plan.PresetName.IsUnknown() && strings.TrimSpace(plan.PresetName.ValueString()) != "" {
		want := strings.TrimSpace(plan.PresetName.ValueString())
		resolved, err := matchNamedID(want, presetsToNamed(list.Presets))
		if err != nil {
			return fmt.Errorf("preset_name: %w", err)
		}
		if presetID > 0 && resolved != presetID {
			return fmt.Errorf("preset_name %q is catalog id %d, but preset_id is %d", want, resolved, presetID)
		}
		if presetName == "" {
			presetName = want
		}
	}

	if presetName == "" && presetID <= 0 {
		return nil
	}

	p, ok := lookupPreset(list.Presets, presetID, presetName)
	if ok {
		if err := validatePlanAgainstCatalogPreset(*plan, p); err != nil {
			return err
		}
	}

	if presetID <= 0 {
		return nil
	}

	ownOS := !plan.OwnOS.IsNull() && !plan.OwnOS.IsUnknown() && plan.OwnOS.ValueBool()
	hasTemplate := !plan.OSTemplate.IsNull() && !plan.OSTemplate.IsUnknown() && strings.TrimSpace(plan.OSTemplate.ValueString()) != ""
	if !ownOS && !hasTemplate && !plan.OSID.IsNull() && !plan.OSID.IsUnknown() && plan.OSID.ValueInt64() > 0 {
		osList, err := r.client.OSList(ctx, invapi.OSListFilter{Location: location, InstanceID: presetID})
		if err != nil {
			return fmt.Errorf("os/list: %w", err)
		}
		ids := activeIDsFromOS(osList.OSList)
		if !catalogHasID(ids, int(plan.OSID.ValueInt64())) {
			return fmt.Errorf("os_id %d is not available for preset_id %d in location %s", plan.OSID.ValueInt64(), presetID, location)
		}
	}

	if !plan.TrafficPlanID.IsNull() && !plan.TrafficPlanID.IsUnknown() && plan.TrafficPlanID.ValueInt64() > 0 {
		tpList, err := r.client.TrafficPlansList(ctx, invapi.TrafficPlansListFilter{Location: location, InstanceID: presetID})
		if err != nil {
			return fmt.Errorf("traffic_plans/list: %w", err)
		}
		ids := activeIDsFromTraffic(tpList.TrafficPlans)
		if !catalogHasID(ids, int(plan.TrafficPlanID.ValueInt64())) {
			return fmt.Errorf("traffic_plan_id %d is not available for preset_id %d in location %s", plan.TrafficPlanID.ValueInt64(), presetID, location)
		}
	}

	if !plan.SoftID.IsNull() && !plan.SoftID.IsUnknown() && plan.SoftID.ValueInt64() > 0 {
		softList, err := r.client.SoftwareList(ctx, invapi.SoftwareListFilter{Location: location, InstanceID: presetID})
		if err != nil {
			return fmt.Errorf("software/list: %w", err)
		}
		ids := activeIDsFromSoftware(softList.Software)
		if !catalogHasID(ids, int(plan.SoftID.ValueInt64())) {
			return fmt.Errorf("soft_id %d is not available for preset_id %d in location %s", plan.SoftID.ValueInt64(), presetID, location)
		}
	}

	return nil
}

func lookupPreset(presets []invapi.Preset, id int, name string) (invapi.Preset, bool) {
	if id > 0 {
		for _, p := range presets {
			if p.ID == id {
				return p, true
			}
		}
	}
	if strings.TrimSpace(name) != "" {
		for _, p := range presets {
			if strings.EqualFold(p.Name, name) {
				return p, true
			}
		}
	}
	return invapi.Preset{}, false
}

func validatePlanAgainstCatalogPreset(plan serverModel, p invapi.Preset) error {
	disks := invapi.DiskCount(p.HDD.String(), p.Description)
	dedicated := p.Dedicated()

	if !plan.DiskMirror.IsNull() && !plan.DiskMirror.IsUnknown() {
		if err := invapi.ValidateDiskMirror(plan.DiskMirror.ValueString(), disks, dedicated); err != nil {
			return err
		}
	}
	if !plan.NoLVM.IsNull() && !plan.NoLVM.IsUnknown() && plan.NoLVM.ValueBool() && !dedicated {
		return fmt.Errorf("no_lvm is only valid on dedicated presets (catalog virtual=0); %s is virtual=%d", p.Name, p.Virtual)
	}
	if !plan.IPv6Block.IsNull() && !plan.IPv6Block.IsUnknown() && plan.IPv6Block.ValueBool() {
		if !dedicated {
			return fmt.Errorf("ipv6_block is only valid on dedicated presets (catalog virtual=0); %s is virtual=%d", p.Name, p.Virtual)
		}
	}
	return nil
}

func presetsToNamed(presets []invapi.Preset) []namedID {
	items := make([]namedID, 0, len(presets))
	for _, p := range presets {
		items = append(items, namedID{ID: p.ID, Name: p.Name})
	}
	return items
}

func activeIDsFromOS(list []invapi.OSEntry) []int {
	var ids []int
	var all []int
	for _, o := range list {
		all = append(all, o.ID)
		if o.Active != 0 {
			ids = append(ids, o.ID)
		}
	}
	if len(ids) == 0 {
		return all
	}
	return ids
}

func activeIDsFromTraffic(list []invapi.TrafficPlan) []int {
	var ids []int
	var all []int
	for _, p := range list {
		all = append(all, p.ID)
		if p.Active != 0 {
			ids = append(ids, p.ID)
		}
	}
	if len(ids) == 0 {
		return all
	}
	return ids
}

func activeIDsFromSoftware(list []invapi.SoftwareEntry) []int {
	var ids []int
	var all []int
	for _, s := range list {
		all = append(all, s.ID)
		if s.Active != 0 {
			ids = append(ids, s.ID)
		}
	}
	if len(ids) == 0 {
		return all
	}
	return ids
}

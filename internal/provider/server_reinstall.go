package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hostkey-cloud/terraform-provider-hostkey/internal/invapi"
)

const defaultUpdateTimeout = 90 * time.Minute

// needsReinstall is true when install-time fields change on an existing server.
// InvAPI reinstall: eq/order_instance with id=<server> and without preset.
func needsReinstall(plan, state serverModel) bool {
	if stringAttrChanged(plan.OSName, state.OSName) {
		return true
	}
	if intAttrChanged(plan.OSID, state.OSID) {
		return true
	}
	if stringAttrChanged(plan.SoftName, state.SoftName) {
		return true
	}
	if intAttrChanged(plan.SoftID, state.SoftID) {
		return true
	}
	if stringAttrChanged(plan.RootPass, state.RootPass) {
		return true
	}
	if stringAttrChanged(plan.SSHKey, state.SSHKey) {
		return true
	}
	if stringAttrChanged(plan.PostInstallScript, state.PostInstallScript) {
		return true
	}
	if stringAttrChanged(plan.OSTemplate, state.OSTemplate) {
		return true
	}
	if boolAttrChanged(plan.OwnOS, state.OwnOS) {
		return true
	}
	if intAttrChanged(plan.RootSize, state.RootSize) {
		return true
	}
	if stringAttrChanged(plan.ReinstallTrigger, state.ReinstallTrigger) &&
		!plan.ReinstallTrigger.IsNull() && plan.ReinstallTrigger.ValueString() != "" {
		return true
	}
	return false
}

func stringAttrChanged(plan, state types.String) bool {
	if plan.IsUnknown() {
		return false
	}
	return !plan.Equal(state)
}

func intAttrChanged(plan, state types.Int64) bool {
	if plan.IsUnknown() {
		return false
	}
	return !plan.Equal(state)
}

func boolAttrChanged(plan, state types.Bool) bool {
	if plan.IsUnknown() {
		return false
	}
	return !plan.Equal(state)
}

func buildReinstallRequest(plan serverModel, serverID int) invapi.OrderInstanceRequest {
	req := buildOrderRequest(plan)
	req.ServerID = serverID
	req.Preset = "" // must not send preset on reinstall
	return req
}

func (r *serverResource) applyReinstall(ctx context.Context, serverID int, plan serverModel) error {
	if err := r.resolveOrderIDs(ctx, &plan); err != nil {
		return fmt.Errorf("catalog resolve: %w", err)
	}

	ownOS := !plan.OwnOS.IsNull() && plan.OwnOS.ValueBool()
	if !ownOS && (plan.OSID.IsNull() || plan.OSID.ValueInt64() == 0) &&
		(plan.OSTemplate.IsNull() || plan.OSTemplate.ValueString() == "") {
		return fmt.Errorf("reinstall requires os_id/os_name (or own_os=true / os_template)")
	}

	// FAQ: refresh session server list before reinstall if id not found.
	if _, err := r.client.EQUpdateServers(ctx); err != nil {
		tflog.Warn(ctx, "eq/update_servers before reinstall failed", map[string]any{"err": err.Error()})
	}

	orderReq := buildReinstallRequest(plan, serverID)
	tflog.Info(ctx, "Starting server reinstall via eq/order_instance", map[string]any{
		"server_id": serverID,
		"os_id":     orderReq.OSID,
		"soft_id":   orderReq.SoftID,
		"location":  orderReq.LocationName,
	})

	orderResp, err := r.client.EQOrderInstance(ctx, orderReq)
	if err != nil {
		return err
	}

	if orderResp.Callback != "" {
		timeout, diags := plan.Timeouts.Update(ctx, defaultUpdateTimeout)
		if diags.HasError() {
			timeout = defaultUpdateTimeout
		}
		_, waitErr := r.client.WaitForCallback(ctx, orderResp.Callback, invapi.WaitOptions{
			PollInterval: pollIntervalFrom(plan),
			Timeout:      timeout,
		})
		if waitErr != nil {
			return fmt.Errorf("reinstall started (callback=%s) but wait failed: %w", orderResp.Callback, waitErr)
		}
	}

	// Keep same id; confirm still visible.
	if orderResp.ID > 0 && orderResp.ID != serverID {
		return fmt.Errorf("reinstall returned unexpected id %d (expected %d)", orderResp.ID, serverID)
	}
	if _, err := r.client.EQShow(ctx, serverID); err != nil {
		return fmt.Errorf("reinstall finished but eq/show failed: %w", err)
	}
	tflog.Info(ctx, "Reinstall finished", map[string]any{
		"server_id": serverID,
		"status":    orderResp.DeployStatus,
	})
	return nil
}

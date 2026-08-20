package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hostkey-cloud/terraform-provider-hostkey/internal/invapi"
)

var (
	_ resource.Resource                = &serverResource{}
	_ resource.ResourceWithImportState = &serverResource{}
	_ resource.ResourceWithModifyPlan  = &serverResource{}
)

const (
	defaultCreateTimeout = 90 * time.Minute
	defaultDeleteTimeout = 30 * time.Minute
	pollInterval         = 15 * time.Second
)

type serverResource struct {
	client *invapi.Client
}

type serverModel struct {
	ID                 types.String   `tfsdk:"id"`
	PresetID           types.Int64    `tfsdk:"preset_id"`
	PresetName         types.String   `tfsdk:"preset_name"`
	LocationName       types.String   `tfsdk:"location_name"`
	OSID               types.Int64    `tfsdk:"os_id"`
	OSName             types.String   `tfsdk:"os_name"`
	SoftID             types.Int64    `tfsdk:"soft_id"`
	SoftName           types.String   `tfsdk:"soft_name"`
	TrafficPlanID      types.Int64    `tfsdk:"traffic_plan_id"`
	TrafficPlanName    types.String   `tfsdk:"traffic_plan_name"`
	Hostname           types.String   `tfsdk:"hostname"`
	RootPass           types.String   `tfsdk:"root_pass"`
	SSHKey             types.String   `tfsdk:"ssh_key"`
	PostInstallScript  types.String   `tfsdk:"post_install_script"`
	DeployPeriod       types.String   `tfsdk:"deploy_period"`
	DeployNotify       types.Bool     `tfsdk:"deploy_notify"`
	OwnOS              types.Bool     `tfsdk:"own_os"`
	RootSize           types.Int64    `tfsdk:"root_size"`
	DiskMirror         types.String   `tfsdk:"disk_mirror"`
	NoLVM              types.Bool     `tfsdk:"no_lvm"`
	IPv6Block          types.Bool     `tfsdk:"ipv6_block"`
	IPv4Amount         types.Int64    `tfsdk:"ipv4_amount"`
	VLAN               types.Int64    `tfsdk:"vlan"`
	PrivateVLAN        types.Int64    `tfsdk:"private_vlan"`
	CustomDomain       types.String   `tfsdk:"custom_domain"`
	OSTemplate         types.String   `tfsdk:"os_template"`
	DeployOptions      types.String   `tfsdk:"deploy_options"`
	ExtraOrderParams   types.Map      `tfsdk:"extra_order_params"`
	Tags               types.Map      `tfsdk:"tags"`
	PollIntervalSecs   types.Int64    `tfsdk:"poll_interval_seconds"`
	MainIPv4           types.String   `tfsdk:"main_ipv4"`
	Status             types.String   `tfsdk:"status"`
	Invoice            types.Int64    `tfsdk:"invoice"`
	CancellationReason types.String   `tfsdk:"cancellation_reason"`
	CancellationType   types.Int64    `tfsdk:"cancellation_type"`
	PowerState         types.String   `tfsdk:"power_state"`
	PowerOffHard       types.Bool     `tfsdk:"power_off_hard"`
	RebootTrigger      types.String   `tfsdk:"reboot_trigger"`
	ReinstallTrigger   types.String   `tfsdk:"reinstall_trigger"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
}

func NewServerResource() resource.Resource {
	return &serverResource{}
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hostkey server ordered via InvAPI eq/order_instance. " +
			"Changing OS/software/root_pass/ssh_key (and related install fields) reinstalls the same server id — data is wiped. " +
			"Preset/location/traffic/billing changes still force replace (new order).",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "InvAPI server ID. While deploy is in progress after a Paid order, Terraform may store pending:<invoice> until apply links the real id.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"preset_id": schema.Int64Attribute{
				Description: "Preset ID. Optional when preset_name is set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					requiresReplaceOnIDChange(),
				},
				Validators: []validator.Int64{
					int64AtLeast("preset_id", 1),
				},
			},
			"preset_name": schema.StringAttribute{
				Description: "Preset name from the catalog (e.g. vm.pico). Looks up preset_id if not set.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location_name": schema.StringAttribute{
				Description: "DC location code: NL, US, FI, DE, RU, etc. (not the same as provider region COM/RU).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					locationCodeValidator(),
				},
			},
			"os_id": schema.Int64Attribute{
				Description: "OS ID. Optional when os_name is set. Changing OS on an existing server triggers reinstall (same id).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64AtLeast("os_id", 1),
				},
			},
			"os_name": schema.StringAttribute{
				Description: "OS name from the catalog (e.g. Ubuntu 22.04). At plan time the provider syncs os_id from this name. Change triggers reinstall on an existing server.",
				Optional:    true,
			},
			"soft_id": schema.Int64Attribute{
				Description: "Marketplace software ID. Optional when soft_name is set. Change triggers reinstall.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64AtLeast("soft_id", 1),
				},
			},
			"soft_name": schema.StringAttribute{
				Description: "Marketplace software name. At plan time the provider syncs soft_id from this name. Change triggers reinstall.",
				Optional:    true,
			},
			"traffic_plan_id": schema.Int64Attribute{
				Description: "Traffic plan ID. Optional when traffic_plan_name is set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					requiresReplaceOnIDChange(),
				},
				Validators: []validator.Int64{
					int64AtLeast("traffic_plan_id", 1),
				},
			},
			"traffic_plan_name": schema.StringAttribute{
				Description: "Traffic plan name from the catalog (e.g. 3 TB / 1 Gbps VM). At plan time the provider syncs traffic_plan_id from this name. Changing preset/location/traffic forces replace (new order).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hostname": schema.StringAttribute{
				Description: "Server hostname.",
				Optional:    true,
				Validators: []validator.String{
					hostnameValidator(),
					stringMaxLen("hostname", maxHostnameLen),
				},
			},
			"root_pass": schema.StringAttribute{
				Description: "Root password (8-30 chars: upper, lower, digit, and one of % - _ +; no @/#). Change triggers reinstall.",
				Required:    true,
				Sensitive:   true,
				Validators: []validator.String{
					rootPassRules(),
				},
			},
			"ssh_key": schema.StringAttribute{
				Description: "Public SSH key injected during deploy/reinstall. Change triggers reinstall.",
				Optional:    true,
				Validators: []validator.String{
					sshPublicKeyValidator(),
				},
			},
			"post_install_script": schema.StringAttribute{
				Description: "Post-install shell script. Change triggers reinstall.",
				Optional:    true,
				Validators: []validator.String{
					stringMaxLen("post_install_script", maxPostInstallScriptLen),
				},
			},
			"deploy_period": schema.StringAttribute{
				Description: "Billing period: hourly, monthly, quarterly, semi-annually, annually. Omit to use InvAPI default.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					oneOfStrings("deploy_period", "hourly", "monthly", "quarterly", "semi-annually", "annually"),
				},
			},
			"deploy_notify": schema.BoolAttribute{
				Description: "Send email when deployment completes. Omit to use InvAPI default.",
				Optional:    true,
			},
			"own_os": schema.BoolAttribute{
				Description: "Skip OS installation during deploy/reinstall. Change triggers reinstall.",
				Optional:    true,
			},
			"root_size": schema.Int64Attribute{
				Description: "Root partition size in GB for the OS volume. Omit to use the full boot disk (panel: 100% of boot disk). Change triggers reinstall.",
				Optional:    true,
				Validators: []validator.Int64{
					int64Between("root_size", minRootSizeGB, maxRootSizeGB),
				},
			},
			"disk_mirror": schema.StringAttribute{
				Description: "Bare-metal disk layout via InvAPI disk_mirror: hba, raid0, raid1, raid10. Validated against presets/list disk count (hdd/description). Omit when the catalog shows 1 disk (panel RAID type is empty; hba is not processed). RAID0/1 need 2+ disks; RAID10 needs 4+. Change triggers reinstall.",
				Optional:    true,
				Validators: []validator.String{
					diskMirrorValidator(),
				},
			},
			"no_lvm": schema.BoolAttribute{
				Description: "Disable LVM and use classic partitions (bare metal only). Maps to InvAPI no_lvm=1. Change triggers reinstall.",
				Optional:    true,
			},
			"ipv6_block": schema.BoolAttribute{
				Description: "Request IPv6 /64 at order time for dedicated presets (catalog virtual=0) in NL/US. InvAPI presets/list does not expose a per-preset IPv6 checkbox; omit unless the order form shows it. Maps to InvAPI ipv6=1. Forces replace.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"ipv4_amount": schema.Int64Attribute{
				Description: "Number of IPv4 addresses to order.",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64Between("ipv4_amount", minIPv4Amount, maxIPv4Amount),
				},
			},
			"vlan": schema.Int64Attribute{
				Description: "Private VLAN ID (InvAPI vlan).",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64AtLeast("vlan", 1),
				},
			},
			"private_vlan": schema.Int64Attribute{
				Description: "Private VLAN ID (InvAPI private_vlan).",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64AtLeast("private_vlan", 1),
				},
			},
			"custom_domain": schema.StringAttribute{
				Description: "Custom domain for the server.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"os_template": schema.StringAttribute{
				Description: "OS template for deploy-from-template (admin/advanced). Change triggers reinstall.",
				Optional:    true,
				Validators: []validator.String{
					stringMaxLen("os_template", maxOSTemplateLen),
				},
			},
			"deploy_options": schema.StringAttribute{
				Description: "InvAPI deploy_options string (billing/location options when required).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringMaxLen("deploy_options", maxDeployOptionsLen),
				},
			},
			"extra_order_params": schema.MapAttribute{
				Description: "Advanced: closed. All eq/order_instance fields are typed attributes; any extra_order_params key is rejected and is not forwarded.",
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.MapAttribute{
				Description: "User tags on the server (tag name → value). Synced via tags/add and tags/remove after the server exists.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"poll_interval_seconds": schema.Int64Attribute{
				Description: "How often to poll deploy status (default 15).",
				Optional:    true,
				Validators: []validator.Int64{
					int64Between("poll_interval_seconds", minPollIntervalSecs, maxPollIntervalSecs),
				},
			},
			"main_ipv4": schema.StringAttribute{
				Description: "Primary IPv4 address after deploy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Last known server status from InvAPI.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"invoice": schema.Int64Attribute{
				Description: "WHMCS invoice id from order_instance (set after Paid).",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"cancellation_reason": schema.StringAttribute{
				Description: "Reason passed to whmcs/request_cancellation on destroy.",
				Optional:    true,
			},
			"cancellation_type": schema.Int64Attribute{
				Description: "Cancellation type on destroy: 0 = end of billing period, 1 = immediate with refund (when allowed). Omit for InvAPI/panel default.",
				Optional:    true,
				Validators: []validator.Int64{
					oneOfInt64("cancellation_type", 0, 1),
				},
			},
			"power_state": schema.StringAttribute{
				Description: `Desired power state: "on" or "off". Maps to eq/on and eq/off (or eq/hard_off when power_off_hard=true). Omit to leave power unmanaged by Terraform.`,
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					oneOfStrings("power_state", "on", "off"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"power_off_hard": schema.BoolAttribute{
				Description: "When power_state is set to off, call eq/hard_off instead of eq/off.",
				Optional:    true,
			},
			"reboot_trigger": schema.StringAttribute{
				Description: "One-shot reboot (eq/reboot): change this string (e.g. timestamp) to reboot on apply. Not a desired-state — value is kept after reboot.",
				Optional:    true,
			},
			"reinstall_trigger": schema.StringAttribute{
				Description: "Force reinstall with the same OS/software: change this string to wipe and reinstall via eq/order_instance (id=server). Data is destroyed.",
				Optional:    true,
			},
		},
	}
}

func (r *serverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*invapi.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected *invapi.Client, got %T", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *serverResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan serverModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state serverModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"Configure the hostkey provider (api_key, region) before planning hostkey_server changes.",
		)
		return
	}
	var config serverModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.resolveOrderIDs(ctx, &plan, &config); err != nil {
		resp.Diagnostics.AddError("Catalog name resolve failed", err.Error())
		return
	}
	if plan.LocationName.IsUnknown() {
		// location_name (and everything resolved from it) cannot be checked against
		// the Hostkey catalog while unknown at plan time. Create/Update re-run
		// catalog checks once it is known.
		resp.Diagnostics.AddAttributeWarning(
			path.Root("location_name"),
			"Catalog validation deferred until apply",
			"location_name is unknown at plan time, so Terraform cannot verify preset/OS/software/traffic-plan availability or root_size against disk capacity right now. If the resolved combination is not available in the Hostkey catalog, apply will fail with a clear error instead of silently placing an invalid order.",
		)
	} else if err := r.verifyOrderCatalog(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Catalog verification failed", err.Error())
		return
	}
	// Keep optional computed ids null/known instead of unknown when names were not requested.
	stabilizeOptionalID(&plan.SoftID, plan.SoftName, state.SoftID)
	stabilizeOptionalID(&plan.TrafficPlanID, plan.TrafficPlanName, state.TrafficPlanID)
	stabilizeOptionalID(&plan.OSID, plan.OSName, state.OSID)
	stabilizeOptionalID(&plan.PresetID, plan.PresetName, state.PresetID)

	resp.Diagnostics.Append(validateServerPlan(ctx, plan, state, req.State.Raw.IsNull())...)

	if !req.State.Raw.IsNull() && needsReinstall(plan, state) {
		resp.Diagnostics.AddWarning(
			"Server reinstall will destroy disk data",
			"This plan changes install-time fields (OS, software, root password, SSH key, disk layout, etc.). Apply runs eq/order_instance reinstall on the same server id — the disk is wiped. This is not a metadata-only update like tags or hostname.",
		)
		// Make the most common destructive change obvious in the diff.
		// Terraform still shows `update in-place` for reinstall, so we use attribute-level warnings.
		if installStringChanged(plan.OSName, state.OSName) {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("os_name"),
				"Changing os_name will reinstall and wipe the disk",
				"Apply runs eq/order_instance reinstall on the same server id; all disk data will be lost.",
			)
		}
	}

	if !req.State.Raw.IsNull() {
		if _, ok := parsePendingInvoice(state.ID.ValueString()); ok {
			// ID must be unknown so apply can replace pending:<invoice> with the real
			// server id without an inconsistent-result error. This stays update-in-place
			// (not replace) unless the resource was previously tainted.
			plan.ID = types.StringUnknown()
			plan.MainIPv4 = types.StringUnknown()
			plan.Status = types.StringUnknown()
			if !state.Invoice.IsNull() && !state.Invoice.IsUnknown() {
				plan.Invoice = state.Invoice
			}
			resp.Diagnostics.AddWarning(
				"Server deploy still in progress",
				fmt.Sprintf("State is %s. Apply will wait for this invoice and will not place a new order. Until the server id is linked, live status is in the Hostkey panel.", state.ID.ValueString()),
			)
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func stabilizeOptionalID(id *types.Int64, name types.String, stateID types.Int64) {
	if id == nil {
		return
	}
	if !id.IsUnknown() {
		return
	}
	if !name.IsNull() && name.ValueString() != "" {
		return // still resolving / will be set
	}
	if !stateID.IsNull() && !stateID.IsUnknown() {
		*id = stateID
		return
	}
	*id = types.Int64Null()
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.resolveOrderIDs(ctx, &plan, &plan); err != nil {
		resp.Diagnostics.AddError("Catalog name resolve failed", err.Error())
		return
	}
	if err := r.verifyOrderCatalog(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Catalog verification failed", err.Error())
		return
	}

	if plan.PresetID.IsNull() && (plan.PresetName.IsNull() || plan.PresetName.ValueString() == "") {
		resp.Diagnostics.AddError("Missing preset", "Set preset_id or preset_name when creating a new server.")
		return
	}
	ownOS := !plan.OwnOS.IsNull() && plan.OwnOS.ValueBool()
	if plan.OSID.IsNull() && !ownOS && (plan.OSTemplate.IsNull() || plan.OSTemplate.ValueString() == "") {
		resp.Diagnostics.AddError("Missing OS", "Set os_id or os_name (unless own_os=true or os_template is set).")
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, defaultCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	interval := pollIntervalFrom(plan)

	orderReq := buildOrderRequest(plan)

	beforeList, err := r.client.EQList(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cannot snapshot existing servers",
			err.Error()+"; refusing to call order_instance without a pre-order eq/list snapshot.",
		)
		return
	}
	known, err := snapshotKnownIDs(beforeList)
	if err != nil {
		resp.Diagnostics.AddError("Cannot snapshot existing servers", err.Error())
		return
	}

	// Additional pre-order snapshot from eq/update_servers.servers.
	// In some accounts eq/list may be stale/incomplete, which can cause
	// previously-existing servers to be treated as "newcomers" later during
	// pending resolution. By unioning both sources we reduce false newcomers
	// and avoid relying on hostname fields from eq/show.
	if upd, updErr := r.client.EQUpdateServers(ctx); updErr == nil {
		if ids, idErr := upd.IDs(); idErr == nil {
			for _, id := range ids {
				known[id] = struct{}{}
			}
		}
	}

	tflog.Info(ctx, "Ordering Hostkey server", map[string]any{
		"preset_id": orderReq.Preset,
		"location":  orderReq.LocationName,
	})

	orderResp, err := r.client.EQOrderInstance(ctx, orderReq)
	if err != nil {
		resp.Diagnostics.AddError("Order failed", err.Error())
		return
	}

	tflog.Info(ctx, "order_instance response", map[string]any{
		"id":           orderResp.ID,
		"callback_set": orderResp.Callback != "",
		"invoice":      orderResp.Invoice,
		"status":       orderResp.Status,
	})

	// CRITICAL: persist partial state immediately after Paid so interrupted apply
	// does not re-enter Create without tracking (and re-order).
	if orderResp.Invoice > 0 {
		pending := plan
		pending.ID = types.StringValue(pendingID(orderResp.Invoice))
		pending.Invoice = types.Int64Value(int64(orderResp.Invoice))
		pending.Status = types.StringValue(orderResp.Status)
		if pending.Status.ValueString() == "" {
			pending.Status = types.StringValue("Pending")
		}
		pending.MainIPv4 = types.StringValue("")
		if pending.PowerState.IsUnknown() {
			pending.PowerState = types.StringNull()
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &pending)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := setPrivateKnownIDs(ctx, resp.Private, known); err != nil {
			resp.Diagnostics.AddWarning("private state", err.Error())
		}
		if err := setPrivateCallback(ctx, resp.Private, orderResp.Callback); err != nil {
			resp.Diagnostics.AddWarning("private state", err.Error())
		}
	}

	serverID := orderResp.ID
	if serverID > 0 {
		if err := acceptNewServerID(serverID, known); err != nil {
			resp.Diagnostics.AddError("Unexpected order id", err.Error())
			return
		}
	}
	if serverID == 0 {
		found, cb, waitErr := r.client.WaitForPendingServer(ctx, orderResp.Invoice, orderResp.Callback, known, plan.Hostname.ValueString(), invapi.WaitOptions{
			PollInterval: interval,
			Timeout:      createTimeout,
			OnPoll: func(status string) {
				tflog.Info(ctx, "waiting for server deploy", map[string]any{
					"invoice": orderResp.Invoice,
					"status":  status,
				})
			},
		})
		if cb != "" {
			if err := setPrivateCallback(ctx, resp.Private, cb); err != nil {
				resp.Diagnostics.AddWarning("private state", err.Error())
			}
		}
		if waitErr != nil {
			resp.Diagnostics.AddWarning(
				"Deploy still in progress",
				fmt.Sprintf("%v; callback=%q invoice=%d. State kept as pending:<invoice>. Re-run apply to wait for this invoice (will not place a new order).", waitErr, cb, orderResp.Invoice),
			)
			return
		}
		if err := acceptNewServerID(found, known); err != nil {
			resp.Diagnostics.AddError("Unexpected new server id", err.Error())
			return
		}
		serverID = found
	}

	state, d := r.readServerState(ctx, serverID, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// eq/order_instance has been observed to intermittently not apply hostname
	// even though the request included it. Detect mismatch and self-heal via
	// eq/rename_server instead of silently persisting a wrong hostname in state.
	if want := strings.TrimSpace(plan.Hostname.ValueString()); !plan.Hostname.IsNull() && want != "" &&
		!strings.EqualFold(want, strings.TrimSpace(state.Hostname.ValueString())) {
		tflog.Warn(ctx, "hostname not applied by order_instance; attempting eq/rename_server", map[string]any{
			"server_id": serverID,
			"want":      want,
			"got":       state.Hostname.ValueString(),
		})
		if err := r.client.EQRenameServer(ctx, serverID, want); err != nil {
			resp.Diagnostics.AddWarning(
				"Hostname was not applied by order_instance and automatic fix-up failed",
				fmt.Sprintf("InvAPI reported hostname %q instead of the requested %q, and the automatic eq/rename_server fix-up failed: %v. The server was created successfully; only the hostname metadata is affected. Fix it manually in the panel or re-apply.", state.Hostname.ValueString(), want, err),
			)
		} else if refreshed, rd := r.readServerState(ctx, serverID, plan); !rd.HasError() {
			resp.Diagnostics.Append(rd...)
			state.Hostname = refreshed.Hostname
			if !strings.EqualFold(strings.TrimSpace(state.Hostname.ValueString()), want) {
				resp.Diagnostics.AddWarning(
					"Hostname still does not match after automatic fix-up",
					fmt.Sprintf("Requested hostname %q; InvAPI now reports %q after eq/rename_server. Verify manually in the panel.", want, state.Hostname.ValueString()),
				)
			}
		} else {
			resp.Diagnostics.Append(rd...)
		}
	}

	// ssh_key cannot be verified via any InvAPI field; it has been observed to
	// intermittently not get written even when order_instance accepted it.
	if !plan.SSHKey.IsNull() && strings.TrimSpace(plan.SSHKey.ValueString()) != "" {
		resp.Diagnostics.AddWarning(
			"Verify SSH key access before relying on it",
			"ssh_key was sent to InvAPI, but eq/show exposes no field confirming the key was written to authorized_keys on the deployed server, and this has been observed to intermittently fail even on an unchanged config (root_pass still applies correctly in that case). Test SSH login with the key before removing other access. If the key is missing, set reinstall_trigger to force a clean reinstall that resends ssh_key, or fall back to root_pass.",
		)
	}

	if orderResp.Invoice > 0 {
		state.Invoice = types.Int64Value(int64(orderResp.Invoice))
	}
	if err := r.syncTags(ctx, serverID, plan.Tags, types.MapNull(types.StringType)); err != nil {
		resp.Diagnostics.AddWarning("Tags", err.Error())
	} else if live, err := r.readUserTags(ctx, serverID); err == nil {
		state.Tags = filterConfiguredTags(plan.Tags, live)
	}
	if err := r.applyPowerState(ctx, serverID, plan, state); err != nil {
		resp.Diagnostics.AddWarning("Power state", err.Error())
	} else if powerStateConfigured(plan) {
		refreshed, rd := r.readServerState(ctx, serverID, plan)
		resp.Diagnostics.Append(rd...)
		if !resp.Diagnostics.HasError() {
			state.PowerState = refreshed.PowerState
			state.Status = refreshed.Status
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if strings.HasPrefix(id, pendingIDPrefix) {
		resolved, cb, err := r.lookupThisPending(ctx, req.Private, state)
		if cb != "" {
			if setErr := setPrivateCallback(ctx, resp.Private, cb); setErr != nil {
				resp.Diagnostics.AddWarning("private state", setErr.Error())
			}
		}
		if err != nil {
			tflog.Info(ctx, "pending server not ready yet", map[string]any{"id": id, "err": err.Error()})
			resp.Diagnostics.AddWarning(
				"Server deploy still in progress",
				fmt.Sprintf("%s. Apply will wait for this invoice (no new order). Live status is in the Hostkey panel until the server id is linked.", err.Error()),
			)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
		newState, d := r.readServerState(ctx, resolved, state)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !state.Invoice.IsNull() {
			newState.Invoice = state.Invoice
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
		return
	}

	serverID, err := strconv.Atoi(id)
	if err != nil {
		resp.Diagnostics.AddError("Invalid server id", err.Error())
		return
	}

	if gone, goneErr := r.serverGone(ctx, serverID); goneErr != nil {
		resp.Diagnostics.AddError("Read server failed", goneErr.Error())
		return
	} else if gone {
		// Without this, a server cancelled/deleted outside Terraform left the
		// resource in state forever with a repeated Read error on every plan.
		resp.State.RemoveResource(ctx)
		return
	}

	newState, d := r.readServerState(ctx, serverID, state)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !state.Invoice.IsNull() {
		newState.Invoice = state.Invoice
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *serverResource) lookupThisPending(ctx context.Context, priv privateData, state serverModel) (int, string, error) {
	invoice, ok := pendingInvoiceFromState(state)
	if !ok {
		return 0, "", fmt.Errorf("invalid pending id %q", state.ID.ValueString())
	}
	known, _ := getPrivateKnownIDs(ctx, priv)
	id, cb, err := r.client.LookupPendingServer(ctx, invoice, getPrivateCallback(ctx, priv), known, state.Hostname.ValueString())
	return id, cb, err
}

func keepPendingComputed(plan *serverModel, state serverModel) {
	plan.ID = state.ID
	plan.Invoice = state.Invoice
	plan.Status = state.Status
	plan.MainIPv4 = state.MainIPv4
	plan.PowerState = state.PowerState
}

func keysInt(m map[int]struct{}) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serverModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if strings.HasPrefix(state.ID.ValueString(), pendingIDPrefix) {
		invoice, ok := pendingInvoiceFromState(state)
		if !ok {
			resp.Diagnostics.AddError("Invalid pending id", state.ID.ValueString())
			return
		}
		waitTimeout, diags := plan.Timeouts.Create(ctx, defaultCreateTimeout)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		known, _ := getPrivateKnownIDs(ctx, req.Private)
		waitCtx, cancel := context.WithTimeout(ctx, waitTimeout)
		defer cancel()
		resolved, cb, err := r.client.WaitForPendingServer(waitCtx, invoice, getPrivateCallback(ctx, req.Private), known, plan.Hostname.ValueString(), invapi.WaitOptions{
			PollInterval: pollIntervalFrom(plan),
			Timeout:      waitTimeout,
			OnPoll: func(status string) {
				tflog.Info(ctx, "waiting for pending server link", map[string]any{
					"invoice": invoice,
					"status":  status,
				})
			},
		})
		if cb != "" {
			if setErr := setPrivateCallback(ctx, resp.Private, cb); setErr != nil {
				resp.Diagnostics.AddWarning("private state", setErr.Error())
			}
		}
		if err != nil {
			keepPendingComputed(&plan, state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			// Warning (not Error): keep apply non-tainted and resume-friendly, same as Create.
			resp.Diagnostics.AddWarning(
				"Deploy still in progress",
				fmt.Sprintf("%v. State kept as %s — re-run apply to wait for this invoice (will not place a new order).", err, state.ID.ValueString()),
			)
			return
		}
		state.ID = types.StringValue(strconv.Itoa(resolved))
	}

	serverID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid server id", err.Error())
		return
	}

	if needsReinstall(plan, state) {
		updateTimeout, diags := plan.Timeouts.Update(ctx, defaultUpdateTimeout)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// If reinstall was started earlier but WaitForCallback failed, resume waiting
		// using the stored callback marker. This prevents a second wipe on the next apply.
		if cb := getPrivateReinstallCallback(ctx, req.Private); cb != "" {
			waitCtx, cancel := context.WithTimeout(ctx, updateTimeout)
			defer cancel()
			_, waitErr := r.client.WaitForCallback(waitCtx, cb, invapi.WaitOptions{
				PollInterval: pollIntervalFrom(plan),
				Timeout:      updateTimeout,
			})
			if waitErr != nil {
				// Persist marker again defensively (best-effort); keep state as-is.
				if err := setPrivateReinstallCallback(ctx, resp.Private, cb); err != nil {
					resp.Diagnostics.AddWarning("private state", err.Error())
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
				resp.Diagnostics.AddError(
					"Reinstall still in progress",
					fmt.Sprintf("%v. State kept as %s — re-run apply to wait for this reinstall (will not place a new order).", waitErr, state.ID.ValueString()),
				)
				return
			}
			// Callback completed; clear marker and continue with readServerState().
			if err := setPrivateReinstallCallback(ctx, resp.Private, ""); err != nil {
				resp.Diagnostics.AddWarning("private state", err.Error())
			}
		} else {
			reCtx, cancel := context.WithTimeout(ctx, updateTimeout)
			defer cancel()
			if err := r.applyReinstall(reCtx, serverID, plan, resp.Private); err != nil {
				var inProg reinstallInProgressError
				if errors.As(err, &inProg) {
					// Keep state as-is so the next apply will resume waiting for the same callback.
					resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
					resp.Diagnostics.AddError(
						"Reinstall still in progress",
						fmt.Sprintf("%v. State kept as %s — re-run apply to wait for this reinstall (will not place a new order).", inProg.cause, state.ID.ValueString()),
					)
					return
				}
				resp.Diagnostics.AddError("Reinstall failed", err.Error())
				return
			}
		}
	}

	if !plan.Hostname.IsNull() && plan.Hostname.ValueString() != state.Hostname.ValueString() {
		if err := r.client.EQRenameServer(ctx, serverID, plan.Hostname.ValueString()); err != nil {
			resp.Diagnostics.AddError("Rename failed", err.Error())
			return
		}
	}

	if err := r.syncTags(ctx, serverID, plan.Tags, state.Tags); err != nil {
		resp.Diagnostics.AddError("Tags sync failed", err.Error())
		return
	}

	if err := r.applyPowerState(ctx, serverID, plan, state); err != nil {
		resp.Diagnostics.AddError("Power state change failed", err.Error())
		return
	}

	if !plan.RebootTrigger.IsNull() &&
		plan.RebootTrigger.ValueString() != "" &&
		plan.RebootTrigger.ValueString() != state.RebootTrigger.ValueString() {
		if err := r.client.EQReboot(ctx, serverID); err != nil {
			resp.Diagnostics.AddError("Reboot failed", err.Error())
			return
		}
		tflog.Info(ctx, "eq/reboot requested", map[string]any{
			"server_id":      serverID,
			"reboot_trigger": plan.RebootTrigger.ValueString(),
		})
	}

	newState, d := r.readServerState(ctx, serverID, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !state.Invoice.IsNull() {
		newState.Invoice = state.Invoice
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serverModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if strings.HasPrefix(state.ID.ValueString(), pendingIDPrefix) {
		resolved, _, err := r.lookupThisPending(ctx, req.Private, state)
		if err != nil {
			inv := state.ID.ValueString()
			if n, ok := pendingInvoiceFromState(state); ok {
				inv = strconv.Itoa(n)
			}
			resp.Diagnostics.AddWarning(
				"Pending server not linked yet",
				fmt.Sprintf("Removed from Terraform state only; WHMCS/InvAPI invoice %s was not cancelled. Cancel that pending service in the Hostkey panel if it is still billed. Terraform could not cancel a server id because this invoice is not linked yet.", inv),
			)
			return
		}
		state.ID = types.StringValue(strconv.Itoa(resolved))
	}

	serverID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid server id", err.Error())
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, defaultDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	reason := "Cancelled via Terraform"
	if !state.CancellationReason.IsNull() {
		reason = state.CancellationReason.ValueString()
	}

	var cancelType *int
	if !state.CancellationType.IsNull() {
		v := int(state.CancellationType.ValueInt64())
		cancelType = &v
	}

	tflog.Info(ctx, "Requesting Hostkey service cancellation", map[string]any{
		"server_id":         serverID,
		"cancellation_type": cancelType,
	})

	if err := r.client.WHMCSRequestCancellation(ctx, serverID, reason, cancelType); err != nil {
		resp.Diagnostics.AddError("Cancellation failed", err.Error())
		return
	}

	// Cancellation is accepted asynchronously; wait until status leaves "rent" when possible.
	deadline := time.Now().Add(deleteTimeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for time.Now().Before(deadline) {
		show, err := r.client.EQShow(ctx, serverID)
		if err != nil {
			return // gone or inaccessible — fine
		}
		st := serverStatus(show)
		if st != "" && !strings.EqualFold(st, "rent") {
			tflog.Info(ctx, "Server left rent status after cancel", map[string]any{"status": st})
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
	resp.Diagnostics.AddWarning(
		"Cancellation submitted",
		"InvAPI accepted request_cancellation, but the server still reports status=rent within the delete timeout. Check the panel; Terraform will still remove it from state.",
	)
}

func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(validateServerImportID(req.ID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *serverResource) readServerState(ctx context.Context, serverID int, template serverModel) (serverModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	show, err := r.client.EQShow(ctx, serverID)
	if err != nil {
		diags.AddError("Read server failed", err.Error())
		return template, diags
	}

	state := template
	state.ID = types.StringValue(strconv.Itoa(serverID))
	state.MainIPv4 = types.StringValue(invapi.MainIPv4(show))
	// Prefer the live hostname from eq/show when available so InvAPI drift
	// (order_instance intermittently skipping hostname) is visible in state.
	if live := invapi.ShowHostname(show); live != "" {
		want := strings.TrimSpace(template.Hostname.ValueString())
		if !template.Hostname.IsNull() && !template.Hostname.IsUnknown() && want != "" && !strings.EqualFold(want, strings.TrimSpace(live)) {
			diags.AddWarning(
				"Live hostname does not match requested hostname",
				fmt.Sprintf("hostname %q was requested, but InvAPI eq/show reports the live server hostname as %q. This has been observed to happen intermittently on Hostkey's side even with an unchanged config. Terraform is recording the live value in state to avoid masking this drift; the next apply will attempt eq/rename_server to fix it, or update hostname in config to %q to accept it.", want, live, live),
			)
		}
		state.Hostname = types.StringValue(live)
	}
	if st := serverStatus(show); st != "" {
		state.Status = types.StringValue(st)
	} else {
		state.Status = types.StringValue(show.Result)
	}
	if ps := invapi.PowerStateFromStatus(state.Status.ValueString()); ps != "" {
		if powerStateConfigured(template) {
			state.PowerState = types.StringValue(ps)
		} else if template.PowerState.IsNull() || template.PowerState.IsUnknown() {
			// Unmanaged: do not invent a value that would create perpetual drift.
			state.PowerState = types.StringNull()
		} else {
			state.PowerState = types.StringValue(ps)
		}
	} else if !powerStateConfigured(template) && (template.PowerState.IsNull() || template.PowerState.IsUnknown()) {
		state.PowerState = types.StringNull()
	}
	if tags, err := r.readUserTags(ctx, serverID); err == nil {
		if !template.Tags.IsNull() {
			state.Tags = filterConfiguredTags(template.Tags, tags)
		}
	}

	return state, diags
}

// serverGone reports whether InvAPI no longer knows about serverID (safe to
// drop from Terraform state), distinguishing that from a transient or auth
// error, which must still surface as a Read error.
func (r *serverResource) serverGone(ctx context.Context, serverID int) (bool, error) {
	_, err := r.client.EQShow(ctx, serverID)
	if err != nil {
		if invapi.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func powerStateConfigured(m serverModel) bool {
	if m.PowerState.IsNull() || m.PowerState.IsUnknown() {
		return false
	}
	s := strings.ToLower(strings.TrimSpace(m.PowerState.ValueString()))
	return s == "on" || s == "off"
}

func (r *serverResource) applyPowerState(ctx context.Context, serverID int, plan, state serverModel) error {
	if !powerStateConfigured(plan) {
		return nil
	}
	want := strings.ToLower(strings.TrimSpace(plan.PowerState.ValueString()))
	have := ""
	if powerStateConfigured(state) {
		have = strings.ToLower(strings.TrimSpace(state.PowerState.ValueString()))
	} else if !state.Status.IsNull() {
		have = invapi.PowerStateFromStatus(state.Status.ValueString())
	}
	if have == want {
		return nil
	}
	switch want {
	case "on":
		return r.client.EQPowerOn(ctx, serverID)
	case "off":
		if !plan.PowerOffHard.IsNull() && plan.PowerOffHard.ValueBool() {
			return r.client.EQHardOff(ctx, serverID)
		}
		return r.client.EQPowerOff(ctx, serverID)
	default:
		return fmt.Errorf("unsupported power_state %q", plan.PowerState.ValueString())
	}
}

func buildOrderRequest(plan serverModel) invapi.OrderInstanceRequest {
	orderReq := invapi.OrderInstanceRequest{
		LocationName: plan.LocationName.ValueString(),
		RootPass:     plan.RootPass.ValueString(),
		OwnOS:        !plan.OwnOS.IsNull() && plan.OwnOS.ValueBool(),
	}
	if !plan.PresetName.IsNull() && plan.PresetName.ValueString() != "" {
		orderReq.Preset = plan.PresetName.ValueString()
	} else if !plan.PresetID.IsNull() {
		orderReq.Preset = strconv.FormatInt(plan.PresetID.ValueInt64(), 10)
	}
	if !plan.OSID.IsNull() {
		orderReq.OSID = int(plan.OSID.ValueInt64())
	}
	if !plan.DeployNotify.IsNull() {
		v := plan.DeployNotify.ValueBool()
		orderReq.DeployNotify = &v
	}
	if !plan.SoftID.IsNull() {
		orderReq.SoftID = int(plan.SoftID.ValueInt64())
	}
	if !plan.TrafficPlanID.IsNull() {
		orderReq.TrafficPlan = int(plan.TrafficPlanID.ValueInt64())
	}
	if !plan.Hostname.IsNull() {
		orderReq.Hostname = plan.Hostname.ValueString()
	}
	if !plan.SSHKey.IsNull() {
		orderReq.SSHKey = plan.SSHKey.ValueString()
	}
	if !plan.PostInstallScript.IsNull() {
		orderReq.PostInstallScript = plan.PostInstallScript.ValueString()
	}
	if !plan.DeployPeriod.IsNull() && plan.DeployPeriod.ValueString() != "" {
		orderReq.DeployPeriod = plan.DeployPeriod.ValueString()
	}
	if !plan.RootSize.IsNull() {
		orderReq.RootSize = int(plan.RootSize.ValueInt64())
	}
	if !plan.DiskMirror.IsNull() && plan.DiskMirror.ValueString() != "" {
		orderReq.DiskMirror = strings.ToLower(strings.TrimSpace(plan.DiskMirror.ValueString()))
	}
	if !plan.NoLVM.IsNull() {
		v := plan.NoLVM.ValueBool()
		orderReq.NoLVM = &v
	}
	if !plan.IPv6Block.IsNull() {
		v := plan.IPv6Block.ValueBool()
		orderReq.IPv6Block = &v
	}
	if !plan.IPv4Amount.IsNull() {
		orderReq.IPv4Amount = int(plan.IPv4Amount.ValueInt64())
	}
	if !plan.VLAN.IsNull() {
		orderReq.VLAN = int(plan.VLAN.ValueInt64())
	}
	if !plan.PrivateVLAN.IsNull() {
		orderReq.PrivateVLAN = int(plan.PrivateVLAN.ValueInt64())
	}
	if !plan.CustomDomain.IsNull() {
		orderReq.CustomDomain = plan.CustomDomain.ValueString()
	}
	if !plan.OSTemplate.IsNull() {
		orderReq.OSTemplate = plan.OSTemplate.ValueString()
	}
	if !plan.DeployOptions.IsNull() {
		orderReq.DeployOptions = plan.DeployOptions.ValueString()
	}
	return orderReq
}

func pollIntervalFrom(plan serverModel) time.Duration {
	if !plan.PollIntervalSecs.IsNull() && plan.PollIntervalSecs.ValueInt64() > 0 {
		return time.Duration(plan.PollIntervalSecs.ValueInt64()) * time.Second
	}
	return pollInterval
}

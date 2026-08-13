package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hkadm/terraform-provider-hostkey/internal/invapi"
)

var (
	_ resource.Resource                = &dnsRecordResource{}
	_ resource.ResourceWithImportState = &dnsRecordResource{}
)

type dnsRecordResource struct {
	client *invapi.Client
}

type dnsRecordModel struct {
	ID       types.String `tfsdk:"id"`
	Zone     types.String `tfsdk:"zone"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Content  types.String `tfsdk:"content"`
	TTL      types.Int64  `tfsdk:"ttl"`
	Priority types.Int64  `tfsdk:"priority"`
}

func NewDNSRecordResource() resource.Resource {
	return &dnsRecordResource{}
}

func (r *dnsRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "DNS record in a Hostkey pdns zone (pdns/add_dns, delete_dns).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic id: zone/name/type/content.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone": schema.StringAttribute{
				Description: "Zone FQDN (must exist as hostkey_dns_domain / pdns zone).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Record name relative to zone (e.g. www or @).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Record type: A, AAAA, CNAME, MX, TXT, NS, …",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Description: "Record value (IP, hostname, text).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl": schema.Int64Attribute{
				Description: "TTL seconds (default InvAPI 3600).",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"priority": schema.Int64Attribute{
				Description: "Priority for MX/SRV.",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *dnsRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*invapi.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *invapi.Client, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func dnsRecordID(zone, name, typ, content string) string {
	return strings.Join([]string{zone, name, strings.ToUpper(typ), content}, "/")
}

func (r *dnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	add := invapi.PDNSAddDNSRequest{
		Zone:    plan.Zone.ValueString(),
		Name:    plan.Name.ValueString(),
		Type:    strings.ToUpper(plan.Type.ValueString()),
		Content: []string{plan.Content.ValueString()},
	}
	if !plan.TTL.IsNull() {
		add.TTL = int(plan.TTL.ValueInt64())
	}
	if !plan.Priority.IsNull() {
		add.Priority = int(plan.Priority.ValueInt64())
	}
	if err := r.client.PDNSAddDNS(ctx, add); err != nil {
		resp.Diagnostics.AddError("Create DNS record failed", err.Error())
		return
	}
	plan.Type = types.StringValue(add.Type)
	plan.ID = types.StringValue(dnsRecordID(add.Zone, add.Name, add.Type, plan.Content.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	records, _, err := r.client.PDNSViewZone(ctx, state.Zone.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read DNS zone failed", err.Error())
		return
	}
	wantName := state.Name.ValueString()
	wantType := strings.ToUpper(state.Type.ValueString())
	wantContent := state.Content.ValueString()
	found := false
	for _, rec := range records {
		name := rec.Name
		if strings.EqualFold(name, wantName) || strings.EqualFold(strings.TrimSuffix(name, "."), wantName) ||
			(wantName == "@" && (name == "@" || name == state.Zone.ValueString() || name == state.Zone.ValueString()+".")) {
			if strings.EqualFold(rec.Type, wantType) && rec.Content == wantContent {
				found = true
				break
			}
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.PDNSDeleteDNS(ctx, state.Zone.ValueString(), state.Name.ValueString(), strings.ToUpper(state.Type.ValueString())); err != nil {
		resp.Diagnostics.AddError("Delete DNS record failed", err.Error())
		return
	}
}

func (r *dnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// zone/name/type/content
	parts := strings.SplitN(req.ID, "/", 4)
	if len(parts) != 4 {
		resp.Diagnostics.AddError("Invalid import id", "Use zone/name/type/content")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content"), parts[3])...)
}

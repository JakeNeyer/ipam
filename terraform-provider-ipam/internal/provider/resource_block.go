package provider

import (
	"context"
	"fmt"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &BlockResource{}
var _ resource.ResourceWithImportState = &BlockResource{}

func NewBlockResource() resource.Resource {
	return &BlockResource{}
}

type BlockResource struct {
	api *client.Client
}

type BlockResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Cidr          types.String `tfsdk:"cidr"`
	TotalIps      types.String `tfsdk:"total_ips"`   // string: derive-only, supports IPv6 /64 etc.
	UsedIps       types.String `tfsdk:"used_ips"`
	AvailableIps  types.String `tfsdk:"available_ips"`
	EnvironmentId types.String `tfsdk:"environment_id"`
}

func (r *BlockResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block"
}

func (r *BlockResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IPAM network block. A block is a CIDR range assigned to an environment; allocations are subnets within a block.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Block UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Block name.",
			},
			"cidr": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "CIDR range (e.g. 10.0.0.0/8).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Environment UUID. Omit for orphaned blocks.",
			},
			"total_ips": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Total IP count in the block (string; supports IPv6 /64 etc.).",
			},
			"used_ips": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "IPs used by allocations.",
			},
			"available_ips": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Available IPs.",
			},
		},
	}
}

func (r *BlockResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	api, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.api = api
}

func (r *BlockResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BlockResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := plan.EnvironmentId.ValueString()
	out, err := r.api.CreateBlock(plan.Name.ValueString(), plan.Cidr.ValueString(), envID)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	r.setModelFromAPI(&plan, out)
	tflog.Trace(ctx, "created ipam_block", map[string]interface{}{"id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BlockResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BlockResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.api.GetBlock(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	r.setModelFromAPI(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BlockResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BlockResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var envID *string
	if !plan.EnvironmentId.IsNull() && plan.EnvironmentId.ValueString() != "" {
		v := plan.EnvironmentId.ValueString()
		envID = &v
	}
	out, err := r.api.UpdateBlock(plan.Id.ValueString(), plan.Name.ValueString(), envID)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	r.setModelFromAPI(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BlockResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BlockResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.api.DeleteBlock(state.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *BlockResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *BlockResource) setModelFromAPI(m *BlockResourceModel, out *client.BlockResponse) {
	m.Id = types.StringValue(out.ID)
	m.Name = types.StringValue(out.Name)
	m.Cidr = types.StringValue(out.CIDR)
	m.EnvironmentId = types.StringValue(out.EnvironmentID)
	m.TotalIps = types.StringValue(out.TotalIPs)
	m.UsedIps = types.StringValue(out.UsedIPs)
	m.AvailableIps = types.StringValue(out.Available)
}

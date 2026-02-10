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

var _ resource.Resource = &AllocationResource{}
var _ resource.ResourceWithImportState = &AllocationResource{}

func NewAllocationResource() resource.Resource {
	return &AllocationResource{}
}

type AllocationResource struct {
	api *client.Client
}

type AllocationResourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	BlockName types.String `tfsdk:"block_name"`
	Cidr      types.String `tfsdk:"cidr"`
}

func (r *AllocationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allocation"
}

func (r *AllocationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IPAM allocation. An allocation is a subnet within a network block (e.g. a VPC or region).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Allocation UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Allocation name.",
			},
			"block_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the parent network block.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cidr": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "CIDR for this allocation (must be within the block).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *AllocationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AllocationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AllocationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.api.CreateAllocation(plan.Name.ValueString(), plan.BlockName.ValueString(), plan.Cidr.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	plan.Id = types.StringValue(out.Id)
	plan.Name = types.StringValue(out.Name)
	plan.BlockName = types.StringValue(out.BlockName)
	plan.Cidr = types.StringValue(out.CIDR)
	tflog.Trace(ctx, "created ipam_allocation", map[string]interface{}{"id": out.Id})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AllocationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AllocationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.api.GetAllocation(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	state.Id = types.StringValue(out.Id)
	state.Name = types.StringValue(out.Name)
	state.BlockName = types.StringValue(out.BlockName)
	state.Cidr = types.StringValue(out.CIDR)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AllocationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AllocationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.api.UpdateAllocation(plan.Id.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	plan.Id = types.StringValue(out.Id)
	plan.Name = types.StringValue(out.Name)
	plan.BlockName = types.StringValue(out.BlockName)
	plan.Cidr = types.StringValue(out.CIDR)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AllocationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AllocationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.api.DeleteAllocation(state.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *AllocationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

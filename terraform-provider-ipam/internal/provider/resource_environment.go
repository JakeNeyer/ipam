package provider

import (
	"context"
	"fmt"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &EnvironmentResource{}
var _ resource.ResourceWithImportState = &EnvironmentResource{}

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

type EnvironmentResource struct {
	api *client.Client
}

type EnvironmentResourceModel struct {
	Id       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Pools    types.List   `tfsdk:"pools"`     // list of { name, cidr }
	PoolIds  types.List   `tfsdk:"pool_ids"` // computed: UUIDs of created pools
}

type poolBlockModel struct {
	Name types.String `tfsdk:"name"`
	Cidr types.String `tfsdk:"cidr"`
}

func (r *EnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *EnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IPAM environment. Environments group network blocks (e.g. prod, staging). Requires at least one pool.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Environment UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Environment name.",
			},
			"pools": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "At least one pool (CIDR range that blocks in this environment draw from).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Pool name.",
						},
						"cidr": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Pool CIDR (e.g. 10.0.0.0/8).",
						},
					},
				},
			},
			"pool_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "UUIDs of pools created with this environment (same order as `pools`).",
			},
		},
	}
}

func (r *EnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var poolBlocks []poolBlockModel
	resp.Diagnostics.Append(plan.Pools.ElementsAs(ctx, &poolBlocks, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	poolList := make([]client.PoolInput, 0, len(poolBlocks))
	for _, pm := range poolBlocks {
		poolList = append(poolList, client.PoolInput{Name: pm.Name.ValueString(), CIDR: pm.Cidr.ValueString()})
	}
	if len(poolList) == 0 {
		resp.Diagnostics.AddError("Invalid config", "at least one pool is required")
		return
	}
	out, err := r.api.CreateEnvironment(plan.Name.ValueString(), poolList)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	plan.Id = types.StringValue(out.Id)
	plan.Name = types.StringValue(out.Name)
	if len(out.PoolIDs) > 0 {
		poolIdVals := make([]types.String, 0, len(out.PoolIDs))
		for _, id := range out.PoolIDs {
			poolIdVals = append(poolIdVals, types.StringValue(id))
		}
		plan.PoolIds, _ = types.ListValueFrom(ctx, types.StringType, poolIdVals)
	}
	tflog.Trace(ctx, "created ipam_environment", map[string]interface{}{"id": out.Id})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.api.GetEnvironment(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	state.Id = types.StringValue(out.Id)
	state.Name = types.StringValue(out.Name)
	poolsResp, err := r.api.ListPools(state.Id.ValueString())
	if err == nil && len(poolsResp.Pools) > 0 {
		objType := types.ObjectType{AttrTypes: map[string]attr.Type{
			"name": types.StringType,
			"cidr": types.StringType,
		}}
		elems := make([]attr.Value, 0, len(poolsResp.Pools))
		poolIdVals := make([]types.String, 0, len(poolsResp.Pools))
		for _, p := range poolsResp.Pools {
			obj, _ := types.ObjectValue(objType.AttrTypes, map[string]attr.Value{
				"name": types.StringValue(p.Name),
				"cidr": types.StringValue(p.CIDR),
			})
			elems = append(elems, obj)
			poolIdVals = append(poolIdVals, types.StringValue(p.ID))
		}
		state.Pools = types.ListValueMust(objType, elems)
		state.PoolIds, _ = types.ListValueFrom(ctx, types.StringType, poolIdVals)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.api.UpdateEnvironment(plan.Id.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	plan.Id = types.StringValue(out.Id)
	plan.Name = types.StringValue(out.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.api.DeleteEnvironment(state.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ReservedBlockResource{}
var _ resource.ResourceWithImportState = &ReservedBlockResource{}

func NewReservedBlockResource() resource.Resource {
	return &ReservedBlockResource{}
}

type ReservedBlockResource struct {
	api *client.Client
}

type ReservedBlockResourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Cidr      types.String `tfsdk:"cidr"`
	Reason    types.String `tfsdk:"reason"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *ReservedBlockResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reserved_block"
}

func (r *ReservedBlockResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reserved CIDR block. Reserved blocks cannot be used as network blocks or allocations (admin only).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Reserved block UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional name for the reserved range.",
			},
			"cidr": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "CIDR range to reserve (e.g. 10.0.0.0/8).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"reason": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional reason for the reservation.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation time (RFC3339).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *ReservedBlockResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ReservedBlockResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ReservedBlockResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := plan.Name.ValueString()
	cidr := strings.TrimSpace(plan.Cidr.ValueString())
	reason := plan.Reason.ValueString()
	out, err := r.api.CreateReservedBlock(name, cidr, reason)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	plan.Id = types.StringValue(out.ID)
	plan.Name = types.StringValue(out.Name)
	plan.Cidr = types.StringValue(out.CIDR)
	plan.Reason = types.StringValue(out.Reason)
	plan.CreatedAt = types.StringValue(out.CreatedAt)
	tflog.Trace(ctx, "created ipam_reserved_block", map[string]interface{}{"id": out.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ReservedBlockResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ReservedBlockResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := r.api.ListReservedBlocks()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	id := state.Id.ValueString()
	for _, b := range list.ReservedBlocks {
		if b.ID == id {
			state.Id = types.StringValue(b.ID)
			state.Name = types.StringValue(b.Name)
			state.Cidr = types.StringValue(b.CIDR)
			state.Reason = types.StringValue(b.Reason)
			state.CreatedAt = types.StringValue(b.CreatedAt)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.Diagnostics.AddError("Reserved block not found", "id: "+id)
}

func (r *ReservedBlockResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported", "ipam_reserved_block does not support in-place updates; change cidr to force replace")
}

func (r *ReservedBlockResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ReservedBlockResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.api.DeleteReservedBlock(state.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
	}
}

func (r *ReservedBlockResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

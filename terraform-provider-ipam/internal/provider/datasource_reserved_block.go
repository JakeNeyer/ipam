package provider

import (
	"context"
	"fmt"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ReservedBlockDataSource{}

func NewReservedBlockDataSource() datasource.DataSource {
	return &ReservedBlockDataSource{}
}

type ReservedBlockDataSource struct {
	api *client.Client
}

type ReservedBlockDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Cidr      types.String `tfsdk:"cidr"`
	Reason    types.String `tfsdk:"reason"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *ReservedBlockDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reserved_block"
}

func (d *ReservedBlockDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single reserved block by ID (admin only).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Reserved block UUID.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Optional name for the reserved range.",
			},
			"cidr": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Reserved CIDR range.",
			},
			"reason": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Optional reason for the reservation.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation time (RFC3339).",
			},
		},
	}
}

func (d *ReservedBlockDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	api, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	d.api = api
}

func (d *ReservedBlockDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ReservedBlockDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := d.api.ListReservedBlocks()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	id := config.Id.ValueString()
	for _, b := range list.ReservedBlocks {
		if b.ID == id {
			config.Id = types.StringValue(b.ID)
			config.Name = types.StringValue(b.Name)
			config.Cidr = types.StringValue(b.CIDR)
			config.Reason = types.StringValue(b.Reason)
			config.CreatedAt = types.StringValue(b.CreatedAt)
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
	}
	resp.Diagnostics.AddError("Not found", "reserved block not found: "+id)
}

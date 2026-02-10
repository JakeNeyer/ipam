package provider

import (
	"context"
	"fmt"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ReservedBlocksDataSource{}

func NewReservedBlocksDataSource() datasource.DataSource {
	return &ReservedBlocksDataSource{}
}

type ReservedBlocksDataSource struct {
	api *client.Client
}

type ReservedBlocksDataSourceModel struct {
	ReservedBlocks []ReservedBlockRefModel `tfsdk:"reserved_blocks"`
}

type ReservedBlockRefModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Cidr      types.String `tfsdk:"cidr"`
	Reason    types.String `tfsdk:"reason"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *ReservedBlocksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reserved_blocks"
}

func (d *ReservedBlocksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List all reserved blocks (admin only).",
		Attributes: map[string]schema.Attribute{
			"reserved_blocks": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of reserved CIDR blocks.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *ReservedBlocksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ReservedBlocksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ReservedBlocksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := d.api.ListReservedBlocks()
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	config.ReservedBlocks = make([]ReservedBlockRefModel, len(out.ReservedBlocks))
	for i, b := range out.ReservedBlocks {
		config.ReservedBlocks[i] = ReservedBlockRefModel{
			Id:        types.StringValue(b.ID),
			Name:      types.StringValue(b.Name),
			Cidr:      types.StringValue(b.CIDR),
			Reason:    types.StringValue(b.Reason),
			CreatedAt: types.StringValue(b.CreatedAt),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

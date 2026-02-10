package provider

import (
	"context"
	"fmt"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AllocationDataSource{}

func NewAllocationDataSource() datasource.DataSource {
	return &AllocationDataSource{}
}

type AllocationDataSource struct {
	api *client.Client
}

type AllocationDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	BlockName types.String `tfsdk:"block_name"`
	Cidr      types.String `tfsdk:"cidr"`
}

func (d *AllocationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allocation"
}

func (d *AllocationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single allocation by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Allocation UUID.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Allocation name.",
			},
			"block_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Parent block name.",
			},
			"cidr": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "CIDR range.",
			},
		},
	}
}

func (d *AllocationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AllocationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AllocationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := d.api.GetAllocation(config.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	config.Id = types.StringValue(out.Id)
	config.Name = types.StringValue(out.Name)
	config.BlockName = types.StringValue(out.BlockName)
	config.Cidr = types.StringValue(out.CIDR)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

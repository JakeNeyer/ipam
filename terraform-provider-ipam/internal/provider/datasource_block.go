package provider

import (
	"context"
	"fmt"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &BlockDataSource{}

func NewBlockDataSource() datasource.DataSource {
	return &BlockDataSource{}
}

type BlockDataSource struct {
	api *client.Client
}

type BlockDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Cidr          types.String `tfsdk:"cidr"`
	TotalIps      types.Int64  `tfsdk:"total_ips"`
	UsedIps       types.Int64  `tfsdk:"used_ips"`
	AvailableIps  types.Int64  `tfsdk:"available_ips"`
	EnvironmentId types.String `tfsdk:"environment_id"`
}

func (d *BlockDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block"
}

func (d *BlockDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single network block by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Block UUID.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Block name.",
			},
			"cidr": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "CIDR range.",
			},
			"total_ips": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total IP count in the block.",
			},
			"used_ips": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "IPs used by allocations.",
			},
			"available_ips": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Available IPs.",
			},
			"environment_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Environment UUID, or empty for orphaned blocks.",
			},
		},
	}
}

func (d *BlockDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BlockDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config BlockDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := d.api.GetBlock(config.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	config.Id = types.StringValue(out.ID)
	config.Name = types.StringValue(out.Name)
	config.Cidr = types.StringValue(out.CIDR)
	config.TotalIps = types.Int64Value(int64(out.TotalIPs))
	config.UsedIps = types.Int64Value(int64(out.UsedIPs))
	config.AvailableIps = types.Int64Value(int64(out.Available))
	config.EnvironmentId = types.StringValue(out.EnvironmentID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

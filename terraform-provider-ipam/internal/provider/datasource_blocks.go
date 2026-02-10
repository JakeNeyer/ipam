package provider

import (
	"context"
	"fmt"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &BlocksDataSource{}

func NewBlocksDataSource() datasource.DataSource {
	return &BlocksDataSource{}
}

type BlocksDataSource struct {
	api *client.Client
}

type BlocksDataSourceModel struct {
	Name          types.String   `tfsdk:"name"`
	EnvironmentId types.String   `tfsdk:"environment_id"`
	OrphanedOnly  types.Bool     `tfsdk:"orphaned_only"`
	Blocks        []BlockRefModel `tfsdk:"blocks"`
}

type BlockRefModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Cidr          types.String `tfsdk:"cidr"`
	TotalIps      types.Int64  `tfsdk:"total_ips"`
	UsedIps       types.Int64  `tfsdk:"used_ips"`
	AvailableIps  types.Int64  `tfsdk:"available_ips"`
	EnvironmentId types.String `tfsdk:"environment_id"`
}

func (d *BlocksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blocks"
}

func (d *BlocksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List network blocks with optional filters.",
		Attributes: map[string]schema.Attribute{
			"name":           schema.StringAttribute{Optional: true, MarkdownDescription: "Filter by name."},
			"environment_id": schema.StringAttribute{Optional: true, MarkdownDescription: "Filter by environment UUID."},
			"orphaned_only":  schema.BoolAttribute{Optional: true, MarkdownDescription: "Only blocks not assigned to an environment."},
			"blocks": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of network blocks matching the filters.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *BlocksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BlocksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config BlocksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := config.EnvironmentId.ValueString()
	orphanedOnly := config.OrphanedOnly.ValueBool()
	out, err := d.api.ListBlocks(config.Name.ValueString(), envID, orphanedOnly, 500, 0)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	config.Blocks = make([]BlockRefModel, len(out.Blocks))
	for i, b := range out.Blocks {
		config.Blocks[i] = BlockRefModel{
			Id:            types.StringValue(b.ID),
			Name:          types.StringValue(b.Name),
			Cidr:          types.StringValue(b.CIDR),
			TotalIps:      types.Int64Value(int64(b.TotalIPs)),
			UsedIps:       types.Int64Value(int64(b.UsedIPs)),
			AvailableIps:  types.Int64Value(int64(b.Available)),
			EnvironmentId: types.StringValue(b.EnvironmentID),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

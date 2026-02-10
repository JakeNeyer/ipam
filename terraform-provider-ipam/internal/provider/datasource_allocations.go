package provider

import (
	"context"
	"fmt"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AllocationsDataSource{}

func NewAllocationsDataSource() datasource.DataSource {
	return &AllocationsDataSource{}
}

type AllocationsDataSource struct {
	api *client.Client
}

type AllocationsDataSourceModel struct {
	Name        types.String           `tfsdk:"name"`
	BlockName   types.String           `tfsdk:"block_name"`
	Allocations []AllocationRefModel   `tfsdk:"allocations"`
}

type AllocationRefModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	BlockName types.String `tfsdk:"block_name"`
	Cidr      types.String `tfsdk:"cidr"`
}

func (d *AllocationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allocations"
}

func (d *AllocationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List allocations with optional filters.",
		Attributes: map[string]schema.Attribute{
			"name":       schema.StringAttribute{Optional: true, MarkdownDescription: "Filter by allocation name."},
			"block_name": schema.StringAttribute{Optional: true, MarkdownDescription: "Filter by block name."},
			"allocations": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of allocations matching the filters.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *AllocationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AllocationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AllocationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := d.api.ListAllocations(config.Name.ValueString(), config.BlockName.ValueString(), 500, 0)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	config.Allocations = make([]AllocationRefModel, len(out.Allocations))
	for i, a := range out.Allocations {
		config.Allocations[i] = AllocationRefModel{
			Id:        types.StringValue(a.Id),
			Name:      types.StringValue(a.Name),
			BlockName: types.StringValue(a.BlockName),
			Cidr:      types.StringValue(a.CIDR),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

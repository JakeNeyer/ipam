package provider

import (
	"context"
	"fmt"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &EnvironmentsDataSource{}

func NewEnvironmentsDataSource() datasource.DataSource {
	return &EnvironmentsDataSource{}
}

type EnvironmentsDataSource struct {
	api *client.Client
}

type EnvironmentsDataSourceModel struct {
	Name         types.String          `tfsdk:"name"`
	Environments []EnvironmentRefModel `tfsdk:"environments"`
}

type EnvironmentRefModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *EnvironmentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environments"
}

func (d *EnvironmentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List IPAM environments with optional name filter.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter by name (substring).",
			},
			"environments": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of environments matching the filter.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Environment UUID.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Environment name.",
						},
					},
				},
			},
		},
	}
}

func (d *EnvironmentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *EnvironmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config EnvironmentsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := d.api.ListEnvironments(config.Name.ValueString(), 500, 0)
	if err != nil {
		resp.Diagnostics.AddError("API error", err.Error())
		return
	}
	config.Environments = make([]EnvironmentRefModel, len(out.Environments))
	for i, e := range out.Environments {
		config.Environments[i] = EnvironmentRefModel{
			Id:   types.StringValue(e.Id),
			Name: types.StringValue(e.Name),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

package provider

import (
	"context"

	"github.com/JakeNeyer/ipam/terraform-provider-ipam/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &IpamProvider{}

type IpamProvider struct {
	version string
}

type IpamProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func (p *IpamProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ipam"
	resp.Version = p.version
}

func (p *IpamProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IPAM provider manages environments, network blocks, allocations, and reserved blocks via the IPAM API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL of the IPAM API (e.g. https://ipam.example.com).",
				Required:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "API token for authentication (Bearer token). Create tokens in the IPAM UI under Admin.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *IpamProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data IpamProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.Endpoint.IsNull() || data.Endpoint.ValueString() == "" {
		resp.Diagnostics.AddError("Missing endpoint", "endpoint is required")
		return
	}
	if data.Token.IsNull() || data.Token.ValueString() == "" {
		resp.Diagnostics.AddError("Missing token", "token is required")
		return
	}
	c, err := client.New(data.Endpoint.ValueString(), data.Token.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *IpamProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewEnvironmentResource,
		NewPoolResource,
		NewReservedBlockResource,
		NewBlockResource,
		NewAllocationResource,
	}
}

func (p *IpamProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewEnvironmentDataSource,
		NewEnvironmentsDataSource,
		NewPoolDataSource,
		NewPoolsDataSource,
		NewReservedBlockDataSource,
		NewReservedBlocksDataSource,
		NewBlockDataSource,
		NewBlocksDataSource,
		NewAllocationDataSource,
		NewAllocationsDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &IpamProvider{version: version}
	}
}

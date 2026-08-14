package vultr

import (
	"context"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*regionsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*regionsDataSource)(nil)
)

type regionsDataSource struct{ client *client.Client }
type regionsDataSourceModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Regions                []regionModel   `tfsdk:"regions"`
	Filters                []filter.Config `tfsdk:"filter"`
}
type regionModel struct {
	ID        types.String `tfsdk:"id"`
	City      types.String `tfsdk:"city"`
	Country   types.String `tfsdk:"country"`
	Continent types.String `tfsdk:"continent"`
}

func NewRegionsDataSource() datasource.DataSource { return &regionsDataSource{} }
func (d *regionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vultr_regions"
}
func (d *regionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Vultr regions for a cloud provider token. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("regions"),
			"regions": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":        schema.StringAttribute{Computed: true, MarkdownDescription: "Region ID."},
				"city":      schema.StringAttribute{Computed: true, MarkdownDescription: "City."},
				"country":   schema.StringAttribute{Computed: true, MarkdownDescription: "Country code."},
				"continent": schema.StringAttribute{Computed: true, MarkdownDescription: "Continent."},
			}}},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}
func (d *regionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureVultrDataSourceClient(req, resp)
}
func (d *regionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config regionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, ok := flex.ReadFilteredTokenList(ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_vultr_regions", "Error listing Vultr regions", resp, d.client.ListVultrRegions,
		func(r client.VultrRegion, field string) (string, bool) {
			switch field {
			case "id":
				return r.ID, true
			case "city":
				return r.City, true
			case "country":
				return r.Country, true
			default:
				return "", false
			}
		})
	if !ok {
		return
	}
	state := regionsDataSourceModel{CloudProviderTokenUUID: config.CloudProviderTokenUUID, Filters: config.Filters, Regions: make([]regionModel, len(items))}
	for i, r := range items {
		state.Regions[i] = regionModel{ID: types.StringValue(r.ID), City: types.StringValue(r.City), Country: types.StringValue(r.Country), Continent: types.StringValue(r.Continent)}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

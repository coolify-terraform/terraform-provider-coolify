package digitalocean

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
	Slug      types.String `tfsdk:"slug"`
	Name      types.String `tfsdk:"name"`
	Available types.Bool   `tfsdk:"available"`
}

func NewRegionsDataSource() datasource.DataSource { return &regionsDataSource{} }

func (d *regionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_digitalocean_regions"
}

func (d *regionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists DigitalOcean regions for a cloud provider token. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("regions"),
			"regions": schema.ListNestedAttribute{
				MarkdownDescription: "The list of DigitalOcean regions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"slug":      schema.StringAttribute{MarkdownDescription: "Region slug (e.g. `nyc1`).", Computed: true},
						"name":      schema.StringAttribute{MarkdownDescription: "Human-readable region name.", Computed: true},
						"available": schema.BoolAttribute{MarkdownDescription: "Whether the region is available.", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *regionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDigitalOceanDataSourceClient(req, resp)
}

func (d *regionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config regionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	regions, ok := flex.ReadFilteredTokenList(
		ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_digitalocean_regions", "Error listing DigitalOcean regions", resp,
		d.client.ListDigitalOceanRegions,
		func(r client.DigitalOceanRegion, field string) (string, bool) {
			switch field {
			case "slug":
				return r.Slug, true
			case "name":
				return r.Name, true
			default:
				return "", false
			}
		},
	)
	if !ok {
		return
	}
	state := regionsDataSourceModel{
		CloudProviderTokenUUID: config.CloudProviderTokenUUID,
		Filters:                config.Filters,
		Regions:                make([]regionModel, len(regions)),
	}
	for i, r := range regions {
		state.Regions[i] = regionModel{
			Slug: types.StringValue(r.Slug), Name: types.StringValue(r.Name), Available: types.BoolValue(r.Available),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

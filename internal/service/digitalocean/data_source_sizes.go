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
	_ datasource.DataSource              = (*sizesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*sizesDataSource)(nil)
)

type sizesDataSource struct{ client *client.Client }

type sizesDataSourceModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Sizes                  []sizeModel     `tfsdk:"sizes"`
	Filters                []filter.Config `tfsdk:"filter"`
}

type sizeModel struct {
	Slug         types.String  `tfsdk:"slug"`
	Memory       types.Int64   `tfsdk:"memory"`
	VCPUs        types.Int64   `tfsdk:"vcpus"`
	Disk         types.Int64   `tfsdk:"disk"`
	PriceMonthly types.Float64 `tfsdk:"price_monthly"`
	Available    types.Bool    `tfsdk:"available"`
}

func NewSizesDataSource() datasource.DataSource { return &sizesDataSource{} }

func (d *sizesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_digitalocean_sizes"
}

func (d *sizesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists DigitalOcean droplet sizes for a cloud provider token. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("sizes"),
			"sizes": schema.ListNestedAttribute{
				MarkdownDescription: "The list of DigitalOcean sizes.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"slug":          schema.StringAttribute{MarkdownDescription: "Size slug (e.g. `s-1vcpu-1gb`).", Computed: true},
						"memory":        schema.Int64Attribute{MarkdownDescription: "Memory in MB.", Computed: true},
						"vcpus":         schema.Int64Attribute{MarkdownDescription: "Number of vCPUs.", Computed: true},
						"disk":          schema.Int64Attribute{MarkdownDescription: "Disk size in GB.", Computed: true},
						"price_monthly": schema.Float64Attribute{MarkdownDescription: "Monthly price in USD.", Computed: true},
						"available":     schema.BoolAttribute{MarkdownDescription: "Whether the size is available.", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *sizesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDigitalOceanDataSourceClient(req, resp)
}

func (d *sizesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sizesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sizes, ok := flex.ReadFilteredTokenList(
		ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_digitalocean_sizes", "Error listing DigitalOcean sizes", resp,
		d.client.ListDigitalOceanSizes,
		func(s client.DigitalOceanSize, field string) (string, bool) {
			switch field {
			case "slug":
				return s.Slug, true
			default:
				return "", false
			}
		},
	)
	if !ok {
		return
	}
	state := sizesDataSourceModel{
		CloudProviderTokenUUID: config.CloudProviderTokenUUID,
		Filters:                config.Filters,
		Sizes:                  make([]sizeModel, len(sizes)),
	}
	for i, s := range sizes {
		state.Sizes[i] = sizeModel{
			Slug: types.StringValue(s.Slug), Memory: types.Int64Value(s.Memory), VCPUs: types.Int64Value(s.VCPUs),
			Disk: types.Int64Value(s.Disk), PriceMonthly: types.Float64Value(s.PriceMonthly), Available: types.BoolValue(s.Available),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

//nolint:dupl // schema and state mapping differ; list/filter logic is in data_source_common.go
package hetzner

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*networksDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*networksDataSource)(nil)
)

type networksDataSource struct {
	client *client.Client
}

type networksDataSourceModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Networks               []networkModel  `tfsdk:"networks"`
	Filters                []filter.Config `tfsdk:"filter"`
}

type networkModel struct {
	ID      types.Int64  `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	IPRange types.String `tfsdk:"ip_range"`
}

// NewNetworksDataSource returns a new Hetzner networks data source instance.
func NewNetworksDataSource() datasource.DataSource { return &networksDataSource{} }

func (d *networksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hetzner_networks"
}

func (d *networksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists existing Hetzner Cloud private networks for a given cloud provider token. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("networks"),
			"networks": schema.ListNestedAttribute{
				MarkdownDescription: "The list of Hetzner private networks.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.Int64Attribute{MarkdownDescription: "The numeric Hetzner network ID.", Computed: true},
						"name":     schema.StringAttribute{MarkdownDescription: "The name of the network.", Computed: true},
						"ip_range": schema.StringAttribute{MarkdownDescription: "The IPv4 range of the network (CIDR).", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *networksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureHetznerDataSourceClient(req, resp)
}

func (d *networksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config networksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networks, ok := readFilteredTokenList(
		ctx,
		config.CloudProviderTokenUUID.ValueString(),
		config.Filters,
		"coolify_hetzner_networks",
		"Error listing Hetzner networks",
		resp,
		d.client,
		d.client.ListHetznerNetworks,
		func(n client.HetznerNetwork, field string) (string, bool) {
			switch field {
			case "id":
				return filter.Int64ToString(n.ID), true
			case "name":
				return n.Name, true
			case "ip_range":
				return n.IPRange, true
			default:
				return "", false
			}
		},
	)
	if !ok {
		return
	}

	state := networksDataSourceModel{
		CloudProviderTokenUUID: config.CloudProviderTokenUUID,
		Filters:                config.Filters,
		Networks:               make([]networkModel, len(networks)),
	}
	for i, n := range networks {
		state.Networks[i] = networkModel{
			ID:      types.Int64Value(n.ID),
			Name:    types.StringValue(n.Name),
			IPRange: types.StringValue(n.IPRange),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

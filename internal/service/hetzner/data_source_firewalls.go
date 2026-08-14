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
	_ datasource.DataSource              = (*firewallsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*firewallsDataSource)(nil)
)

type firewallsDataSource struct {
	client *client.Client
}

type firewallsDataSourceModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Firewalls              []firewallModel `tfsdk:"firewalls"`
	Filters                []filter.Config `tfsdk:"filter"`
}

type firewallModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// NewFirewallsDataSource returns a new Hetzner firewalls data source instance.
func NewFirewallsDataSource() datasource.DataSource { return &firewallsDataSource{} }

func (d *firewallsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hetzner_firewalls"
}

func (d *firewallsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists existing Hetzner Cloud firewalls for a given cloud provider token. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("firewalls"),
			"firewalls": schema.ListNestedAttribute{
				MarkdownDescription: "The list of Hetzner firewalls.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.Int64Attribute{MarkdownDescription: "The numeric Hetzner firewall ID.", Computed: true},
						"name": schema.StringAttribute{MarkdownDescription: "The name of the firewall.", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *firewallsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureHetznerDataSourceClient(req, resp)
}

func (d *firewallsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config firewallsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewalls, ok := readFilteredTokenList(
		ctx,
		config.CloudProviderTokenUUID.ValueString(),
		config.Filters,
		"coolify_hetzner_firewalls",
		"Error listing Hetzner firewalls",
		resp,
		d.client,
		d.client.ListHetznerFirewalls,
		func(fw client.HetznerFirewall, field string) (string, bool) {
			switch field {
			case "id":
				return filter.Int64ToString(fw.ID), true
			case "name":
				return fw.Name, true
			default:
				return "", false
			}
		},
	)
	if !ok {
		return
	}

	state := firewallsDataSourceModel{
		CloudProviderTokenUUID: config.CloudProviderTokenUUID,
		Filters:                config.Filters,
		Firewalls:              make([]firewallModel, len(firewalls)),
	}
	for i, fw := range firewalls {
		state.Firewalls[i] = firewallModel{
			ID:   types.Int64Value(fw.ID),
			Name: types.StringValue(fw.Name),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

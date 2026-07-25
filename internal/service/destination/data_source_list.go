package destination

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*destinationListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*destinationListDataSource)(nil)
)

type destinationListDataSource struct{ client *client.Client }

type destinationListDataSourceModel struct {
	Destinations []destinationItemModel `tfsdk:"destinations"`
	Filters      []filter.Config        `tfsdk:"filter"`
}

type destinationItemModel struct {
	UUID       types.String `tfsdk:"uuid"`
	ServerUUID types.String `tfsdk:"server_uuid"`
	Name       types.String `tfsdk:"name"`
	Network    types.String `tfsdk:"network"`
	Type       types.String `tfsdk:"type"`
}

func NewListDataSource() datasource.DataSource { return &destinationListDataSource{} }

func (d *destinationListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destinations"
}

func (d *destinationListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Coolify destinations for the authenticated team. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"destinations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":        schema.StringAttribute{Computed: true, MarkdownDescription: "Destination UUID."},
						"server_uuid": schema.StringAttribute{Computed: true, MarkdownDescription: "Owning server UUID."},
						"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Display name."},
						"network":     schema.StringAttribute{Computed: true, MarkdownDescription: "Docker network name."},
						"type":        schema.StringAttribute{Computed: true, MarkdownDescription: "`standalone` or `swarm`."},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *destinationListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *destinationListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config destinationListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_destinations"})
	items, err := d.client.ListDestinations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing destinations", fmt.Sprintf("Could not list destinations: %s", err))
		return
	}
	items = filter.Apply(ctx, items, config.Filters, func(dest client.Destination, field string) (string, bool) {
		switch field {
		case "uuid":
			return dest.UUID, true
		case "name":
			return dest.Name, true
		case "network":
			return dest.Network, true
		case "type":
			return dest.Type, true
		case "server_uuid":
			return dest.ServerUUID, true
		default:
			return "", false
		}
	})
	state := destinationListDataSourceModel{Filters: config.Filters, Destinations: make([]destinationItemModel, 0, len(items))}
	for _, dest := range items {
		state.Destinations = append(state.Destinations, destinationItemModel{
			UUID: types.StringValue(dest.UUID), ServerUUID: types.StringValue(dest.ServerUUID),
			Name: flex.StringToFramework(dest.Name), Network: types.StringValue(dest.Network), Type: types.StringValue(dest.Type),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

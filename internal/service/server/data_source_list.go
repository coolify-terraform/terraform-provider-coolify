package server

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var (
	_ datasource.DataSource              = &serversDataSource{}
	_ datasource.DataSourceWithConfigure = &serversDataSource{}
)

type serversDataSource struct {
	client *client.Client
}

type serversDataSourceModel struct {
	Servers []serverDataSourceModel `tfsdk:"servers"`
	Filters []filter.Config         `tfsdk:"filter"`
}

// NewListDataSource returns a new data source that lists all servers.
func NewListDataSource() datasource.DataSource {
	return &serversDataSource{}
}

func (d *serversDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_servers"
}

func (d *serversDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of all Coolify servers.",
		Attributes: map[string]schema.Attribute{
			"servers": schema.ListNestedAttribute{
				MarkdownDescription: "The list of servers.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: serverDataSourceAttributes(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *serversDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *serversDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serversDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	servers, err := d.client.ListServers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing servers", err.Error())
		return
	}

	servers = filter.Apply(servers, config.Filters, func(s client.Server, field string) (string, bool) {
		switch field {
		case "uuid":
			return s.UUID, true
		case "name":
			return s.Name, true
		case "description":
			return s.Description, true
		case "ip":
			return s.IP, true
		case "port":
			return filter.IntToString(s.Port), true
		case "user":
			return s.User, true
		case "is_build_server":
			return filter.BoolToString(s.IsBuildServer), true
		case "is_reachable":
			return filter.BoolToString(s.IsReachable), true
		case "is_usable":
			return filter.BoolToString(s.IsUsable), true
		default:
			return "", false
		}
	})

	var state serversDataSourceModel
	state.Filters = config.Filters
	for _, srv := range servers {
		state.Servers = append(state.Servers, flattenServerDataSourceModel(srv))
	}

	if state.Servers == nil {
		state.Servers = []serverDataSourceModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

package server

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the server.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the server.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "A description of the server.",
							Computed:            true,
						},
						"ip": schema.StringAttribute{
							MarkdownDescription: "The IP address of the server.",
							Computed:            true,
						},
						"port": schema.Int64Attribute{
							MarkdownDescription: "The SSH port of the server.",
							Computed:            true,
						},
						"user": schema.StringAttribute{
							MarkdownDescription: "The SSH user for connecting to the server.",
							Computed:            true,
						},
						"private_key_uuid": schema.StringAttribute{
							MarkdownDescription: "The UUID of the private key used for SSH authentication.",
							Computed:            true,
						},
						"is_build_server": schema.BoolAttribute{
							MarkdownDescription: "Whether this server is used for building applications.",
							Computed:            true,
						},
						"is_reachable": schema.BoolAttribute{
							MarkdownDescription: "Whether the server is currently reachable.",
							Computed:            true,
						},
						"is_usable": schema.BoolAttribute{
							MarkdownDescription: "Whether the server is currently usable for deployments.",
							Computed:            true,
						},
					},
				},
			},
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
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *serversDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	servers, err := d.client.ListServers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing servers", err.Error())
		return
	}

	var state serversDataSourceModel
	for _, srv := range servers {
		state.Servers = append(state.Servers, serverDataSourceModel{
			UUID:           types.StringValue(srv.UUID),
			Name:           types.StringValue(srv.Name),
			Description:    types.StringValue(srv.Description),
			IP:             types.StringValue(srv.IP),
			Port:           types.Int64Value(int64(srv.Port)),
			User:           types.StringValue(srv.User),
			PrivateKeyUUID: types.StringValue(srv.PrivateKeyUUID),
			IsBuildServer:  types.BoolValue(srv.IsBuildServer),
			IsReachable:    types.BoolValue(srv.IsReachable),
			IsUsable:       types.BoolValue(srv.IsUsable),
		})
	}

	if state.Servers == nil {
		state.Servers = []serverDataSourceModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

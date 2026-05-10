package server

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &serverDataSource{}
	_ datasource.DataSourceWithConfigure = &serverDataSource{}
)

type serverDataSource struct {
	client *client.Client
}

type serverDataSourceModel struct {
	UUID           types.String `tfsdk:"uuid"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	IP             types.String `tfsdk:"ip"`
	Port           types.Int64  `tfsdk:"port"`
	User           types.String `tfsdk:"user"`
	PrivateKeyUUID types.String `tfsdk:"private_key_uuid"`
	IsBuildServer  types.Bool   `tfsdk:"is_build_server"`
	IsReachable    types.Bool   `tfsdk:"is_reachable"`
	IsUsable       types.Bool   `tfsdk:"is_usable"`
}

// NewDataSource returns a new server data source.
func NewDataSource() datasource.DataSource {
	return &serverDataSource{}
}

func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Coolify server.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the server.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
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
	}
}

func (d *serverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	srv, err := d.client.GetServer(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading server", err.Error())
		return
	}

	config.UUID = types.StringValue(srv.UUID)
	config.Name = types.StringValue(srv.Name)
	config.Description = types.StringValue(srv.Description)
	config.IP = types.StringValue(srv.IP)
	config.Port = types.Int64Value(int64(srv.Port))
	config.User = types.StringValue(srv.User)
	config.PrivateKeyUUID = types.StringValue(srv.PrivateKeyUUID)
	config.IsBuildServer = types.BoolValue(srv.IsBuildServer)
	config.IsReachable = types.BoolValue(srv.IsReachable)
	config.IsUsable = types.BoolValue(srv.IsUsable)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

package server

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
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
	UUID                                 types.String `tfsdk:"uuid"`
	Name                                 types.String `tfsdk:"name"`
	Description                          types.String `tfsdk:"description"`
	IP                                   types.String `tfsdk:"ip"`
	Port                                 types.Int64  `tfsdk:"port"`
	User                                 types.String `tfsdk:"user"`
	IsBuildServer                        types.Bool   `tfsdk:"is_build_server"`
	IsReachable                          types.Bool   `tfsdk:"is_reachable"`
	IsUsable                             types.Bool   `tfsdk:"is_usable"`
	ConcurrentBuilds                     types.Int64  `tfsdk:"concurrent_builds"`
	DynamicTimeout                       types.Int64  `tfsdk:"dynamic_timeout"`
	DeploymentQueueLimit                 types.Int64  `tfsdk:"deployment_queue_limit"`
	ServerDiskUsageNotificationThreshold types.Int64  `tfsdk:"server_disk_usage_notification_threshold"`
	ServerDiskUsageCheckFrequency        types.String `tfsdk:"server_disk_usage_check_frequency"`
}

func serverDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
		"concurrent_builds": schema.Int64Attribute{
			MarkdownDescription: "How many deployments can run in parallel on this server.",
			Computed:            true,
		},
		"dynamic_timeout": schema.Int64Attribute{
			MarkdownDescription: "Deployment timeout in seconds.",
			Computed:            true,
		},
		"deployment_queue_limit": schema.Int64Attribute{
			MarkdownDescription: "Maximum number of queued deployments. 0 means unlimited.",
			Computed:            true,
		},
		"server_disk_usage_notification_threshold": schema.Int64Attribute{
			MarkdownDescription: "Disk usage percentage at which a notification is sent.",
			Computed:            true,
		},
		"server_disk_usage_check_frequency": schema.StringAttribute{
			MarkdownDescription: "Cron expression for how often disk usage is checked.",
			Computed:            true,
		},
	}
}

func flattenServerDataSourceModel(srv client.Server) serverDataSourceModel {
	model := serverDataSourceModel{
		UUID:          types.StringValue(srv.UUID),
		Name:          types.StringValue(srv.Name),
		Description:   flex.StringToFramework(srv.Description),
		IP:            types.StringValue(srv.IP),
		Port:          types.Int64Value(int64(srv.Port)),
		User:          types.StringValue(srv.User),
		IsBuildServer: types.BoolValue(srv.IsBuildServer),
		IsReachable:   types.BoolValue(srv.IsReachable),
		IsUsable:      types.BoolValue(srv.IsUsable),
	}

	if srv.Settings == nil {
		return model
	}

	model.ConcurrentBuilds = types.Int64Value(int64(srv.Settings.ConcurrentBuilds))
	model.DynamicTimeout = types.Int64Value(int64(srv.Settings.DynamicTimeout))
	model.DeploymentQueueLimit = types.Int64Value(int64(srv.Settings.DeploymentQueueLimit))
	model.ServerDiskUsageNotificationThreshold = types.Int64Value(int64(srv.Settings.ServerDiskUsageNotificationThreshold))
	model.ServerDiskUsageCheckFrequency = flex.StringToFramework(srv.Settings.ServerDiskUsageCheckFrequency)

	return model
}

// NewDataSource returns a new server data source.
func NewDataSource() datasource.DataSource {
	return &serverDataSource{}
}

func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := serverDataSourceAttributes()
	attributes["uuid"] = schema.StringAttribute{
		MarkdownDescription: "The unique identifier of the server.",
		Required:            true,
		Validators:          []validator.String{validate.UUID()},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Coolify server.",
		Attributes:          attributes,
	}
}

func (d *serverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	config = flattenServerDataSourceModel(*srv)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

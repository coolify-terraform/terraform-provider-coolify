package database

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*databaseDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*databaseDataSource)(nil)
)

type databaseDataSource struct {
	client *client.Client
}

type databaseDataSourceModel struct {
	UUID                   types.String `tfsdk:"uuid"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	Type                   types.String `tfsdk:"type"`
	Image                  types.String `tfsdk:"image"`
	IsPublic               types.Bool   `tfsdk:"is_public"`
	PublicPort             types.Int64  `tfsdk:"public_port"`
	ServerUUID             types.String `tfsdk:"server_uuid"`
	ProjectUUID            types.String `tfsdk:"project_uuid"`
	EnvironmentName        types.String `tfsdk:"environment_name"`
	IsLogDrainEnabled      types.Bool   `tfsdk:"is_log_drain_enabled"`
	IsIncludeTimestamps    types.Bool   `tfsdk:"is_include_timestamps"`
	HealthCheckEnabled     types.Bool   `tfsdk:"health_check_enabled"`
	HealthCheckInterval    types.Int64  `tfsdk:"health_check_interval"`
	HealthCheckTimeout     types.Int64  `tfsdk:"health_check_timeout"`
	HealthCheckRetries     types.Int64  `tfsdk:"health_check_retries"`
	HealthCheckStartPeriod types.Int64  `tfsdk:"health_check_start_period"`
	EnableSSL              types.Bool   `tfsdk:"enable_ssl"`
	SSLMode                types.String `tfsdk:"ssl_mode"`
	Status                 types.String `tfsdk:"status"`
	InternalDBUrl          types.String `tfsdk:"internal_db_url"`
}

func NewDataSource() datasource.DataSource {
	return &databaseDataSource{}
}

func (d *databaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (d *databaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to retrieve information about a Coolify database by UUID.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the database.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the database.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the database.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the database (e.g., postgresql, mysql, redis).",
				Computed:            true,
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "The Docker image used by the database.",
				Computed:            true,
			},
			"is_public": schema.BoolAttribute{
				MarkdownDescription: "Whether the database is publicly accessible.",
				Computed:            true,
			},
			"public_port": schema.Int64Attribute{
				MarkdownDescription: "The public port for the database, if publicly accessible.",
				Computed:            true,
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server the database is deployed on.",
				Computed:            true,
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project this database belongs to.",
				Computed:            true,
			},
			"environment_name": schema.StringAttribute{
				MarkdownDescription: "The environment name.",
				Computed:            true,
			},
			"is_log_drain_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether log drain is enabled for this database.",
				Computed:            true,
			},
			"is_include_timestamps": schema.BoolAttribute{
				MarkdownDescription: "Whether timestamps are included in log output.",
				Computed:            true,
			},
			"health_check_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the Docker health check probe is enabled.",
				Computed:            true,
			},
			"health_check_interval": schema.Int64Attribute{
				MarkdownDescription: "Health check interval in seconds.",
				Computed:            true,
			},
			"health_check_timeout": schema.Int64Attribute{
				MarkdownDescription: "Health check timeout in seconds.",
				Computed:            true,
			},
			"health_check_retries": schema.Int64Attribute{
				MarkdownDescription: "Number of consecutive failures before unhealthy.",
				Computed:            true,
			},
			"health_check_start_period": schema.Int64Attribute{
				MarkdownDescription: "Grace period in seconds before health checks count.",
				Computed:            true,
			},
			"enable_ssl": schema.BoolAttribute{
				MarkdownDescription: "Whether SSL/TLS is enabled for database connections.",
				Computed:            true,
			},
			"ssl_mode": schema.StringAttribute{
				MarkdownDescription: "The SSL connection mode (e.g., `require`, `verify-full`). Empty when SSL is not enabled or not supported by the database type.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the database (e.g., running, exited).",
				Computed:            true,
			},
			"internal_db_url": schema.StringAttribute{
				MarkdownDescription: "Internal connection URL for the database, accessible from other containers on the same server. Contains credentials; requires an API token with sensitive-data read permission.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (d *databaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *databaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config databaseDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_database"})

	db, err := d.client.GetDatabase(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading database", fmt.Sprintf("database %s: %s", config.UUID.ValueString(), err))
		return
	}

	config.UUID = types.StringValue(db.UUID)
	config.Name = types.StringValue(db.Name)
	config.Description = flex.StringToFramework(db.Description)
	config.Type = types.StringValue(db.Type)
	config.Image = flex.StringToFramework(db.Image)
	config.IsPublic = types.BoolValue(db.IsPublic)
	config.PublicPort = flex.Int64PtrToFramework(db.PublicPort)
	config.ServerUUID = flex.StringToFramework(db.ServerUUID)
	config.ProjectUUID = flex.StringToFramework(db.ProjectUUID)
	config.EnvironmentName = flex.StringToFramework(db.EnvironmentName)
	config.IsLogDrainEnabled = types.BoolValue(db.IsLogDrainEnabled)
	config.IsIncludeTimestamps = types.BoolValue(db.IsIncludeTimestamps)
	if db.HealthCheckEnabled != nil {
		config.HealthCheckEnabled = types.BoolValue(*db.HealthCheckEnabled)
	}
	config.HealthCheckInterval = flex.Int64PtrToFramework(db.HealthCheckInterval)
	config.HealthCheckTimeout = flex.Int64PtrToFramework(db.HealthCheckTimeout)
	config.HealthCheckRetries = flex.Int64PtrToFramework(db.HealthCheckRetries)
	config.HealthCheckStartPeriod = flex.Int64PtrToFramework(db.HealthCheckStartPeriod)
	config.EnableSSL = types.BoolValue(db.EnableSSL)
	config.SSLMode = flex.StringToFramework(db.SSLMode)
	config.Status = flex.StringToFramework(db.Status)
	config.InternalDBUrl = flex.StringToFramework(db.InternalDBUrl)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

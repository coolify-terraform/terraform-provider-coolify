package database

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
	UUID                types.String `tfsdk:"uuid"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Type                types.String `tfsdk:"type"`
	Image               types.String `tfsdk:"image"`
	IsPublic            types.Bool   `tfsdk:"is_public"`
	PublicPort          types.Int64  `tfsdk:"public_port"`
	ServerUUID          types.String `tfsdk:"server_uuid"`
	ProjectUUID         types.String `tfsdk:"project_uuid"`
	EnvironmentName     types.String `tfsdk:"environment_name"`
	IsLogDrainEnabled   types.Bool   `tfsdk:"is_log_drain_enabled"`
	IsIncludeTimestamps types.Bool   `tfsdk:"is_include_timestamps"`
	EnableSSL           types.Bool   `tfsdk:"enable_ssl"`
	SSLMode             types.String `tfsdk:"ssl_mode"`
	Status              types.String `tfsdk:"status"`
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
		},
	}
}

func (d *databaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	config.EnableSSL = types.BoolValue(db.EnableSSL)
	config.SSLMode = flex.StringToFramework(db.SSLMode)
	config.Status = flex.StringToFramework(db.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

package database

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*databaseListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*databaseListDataSource)(nil)
)

// databaseListDataSource is the data source implementation for listing all Coolify databases.
type databaseListDataSource struct {
	client *client.Client
}

// databaseListDataSourceModel maps the data source schema data.
type databaseListDataSourceModel struct {
	Databases []databaseItemModel `tfsdk:"databases"`
	Filters   []filter.Config     `tfsdk:"filter"`
}

// databaseItemModel maps a single database in the list.
type databaseItemModel struct {
	UUID                types.String `tfsdk:"uuid"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Type                types.String `tfsdk:"type"`
	Image               types.String `tfsdk:"image"`
	IsPublic            types.Bool   `tfsdk:"is_public"`
	IsLogDrainEnabled   types.Bool   `tfsdk:"is_log_drain_enabled"`
	IsIncludeTimestamps types.Bool   `tfsdk:"is_include_timestamps"`
	EnableSSL           types.Bool   `tfsdk:"enable_ssl"`
	SSLMode             types.String `tfsdk:"ssl_mode"`
	Status              types.String `tfsdk:"status"`
}

// NewListDataSource returns a new databases list data source instance.
func NewListDataSource() datasource.DataSource {
	return &databaseListDataSource{}
}

func (d *databaseListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_databases"
}

func (d *databaseListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify databases.",
		Attributes: map[string]schema.Attribute{
			"databases": schema.ListNestedAttribute{
				MarkdownDescription: "The list of databases.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the database.",
							Computed:            true,
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
							MarkdownDescription: "The SSL connection mode.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The current status of the database.",
							Computed:            true,
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *databaseListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *databaseListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config databaseListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_databases"})

	databases, err := d.client.ListDatabases(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing databases", fmt.Sprintf("Could not list databases: %s", err))
		return
	}

	databases = filter.Apply(ctx, databases, config.Filters, func(db client.Database, field string) (string, bool) {
		switch field {
		case "uuid":
			return db.UUID, true
		case "name":
			return db.Name, true
		case "description":
			return db.Description, true
		case "type":
			return db.Type, true
		case "image":
			return db.Image, true
		case "is_public":
			return filter.BoolToString(db.IsPublic), true
		case "is_log_drain_enabled":
			return filter.BoolToString(db.IsLogDrainEnabled), true
		case "is_include_timestamps":
			return filter.BoolToString(db.IsIncludeTimestamps), true
		case "enable_ssl":
			return filter.BoolToString(db.EnableSSL), true
		case "ssl_mode":
			return db.SSLMode, true
		case "status":
			return db.Status, true
		default:
			return "", false
		}
	})

	var state databaseListDataSourceModel
	state.Filters = config.Filters
	for _, db := range databases {
		item := databaseItemModel{
			UUID:                types.StringValue(db.UUID),
			Name:                types.StringValue(db.Name),
			Type:                types.StringValue(db.Type),
			IsPublic:            types.BoolValue(db.IsPublic),
			IsLogDrainEnabled:   types.BoolValue(db.IsLogDrainEnabled),
			IsIncludeTimestamps: types.BoolValue(db.IsIncludeTimestamps),
			EnableSSL:           types.BoolValue(db.EnableSSL),
		}
		item.Description = flex.StringToFramework(db.Description)
		item.Image = flex.StringToFramework(db.Image)
		item.SSLMode = flex.StringToFramework(db.SSLMode)
		item.Status = flex.StringToFramework(db.Status)
		state.Databases = append(state.Databases, item)
	}

	if state.Databases == nil {
		state.Databases = []databaseItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

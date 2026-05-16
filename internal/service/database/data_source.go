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
)

var (
	_ datasource.DataSource              = (*databaseDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*databaseDataSource)(nil)
)

type databaseDataSource struct {
	client *client.Client
}

type databaseDataSourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Type            types.String `tfsdk:"type"`
	Image           types.String `tfsdk:"image"`
	IsPublic        types.Bool   `tfsdk:"is_public"`
	PublicPort      types.Int64  `tfsdk:"public_port"`
	ServerUUID      types.String `tfsdk:"server_uuid"`
	ProjectUUID     types.String `tfsdk:"project_uuid"`
	EnvironmentName types.String `tfsdk:"environment_name"`
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
				MarkdownDescription: "The type of the database (e.g. postgresql, mysql, redis).",
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

	db, err := d.client.GetDatabase(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading database", err.Error())
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

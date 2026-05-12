package database

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
}

// databaseItemModel maps a single database in the list.
type databaseItemModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Image       types.String `tfsdk:"image"`
	IsPublic    types.Bool   `tfsdk:"is_public"`
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
					},
				},
			},
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
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *databaseListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	databases, err := d.client.ListDatabases(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing databases", fmt.Sprintf("Could not list databases: %s", err))
		return
	}

	var state databaseListDataSourceModel
	for _, db := range databases {
		item := databaseItemModel{
			UUID:     types.StringValue(db.UUID),
			Name:     types.StringValue(db.Name),
			Type:     types.StringValue(db.Type),
			IsPublic: types.BoolValue(db.IsPublic),
		}
		item.Description = flex.StringToFramework(db.Description)
		item.Image = flex.StringToFramework(db.Image)
		state.Databases = append(state.Databases, item)
	}

	if state.Databases == nil {
		state.Databases = []databaseItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

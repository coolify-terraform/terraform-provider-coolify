package project

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
	_ datasource.DataSource              = (*projectListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*projectListDataSource)(nil)
)

// projectListDataSource is the data source implementation for listing all Coolify projects.
type projectListDataSource struct {
	client *client.Client
}

// projectListDataSourceModel maps the data source schema data.
type projectListDataSourceModel struct {
	Projects []projectItemModel `tfsdk:"projects"`
}

// projectItemModel maps a single project in the list.
type projectItemModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

// NewListDataSource returns a new projects list data source instance.
func NewListDataSource() datasource.DataSource {
	return &projectListDataSource{}
}

func (d *projectListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *projectListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify projects.",
		Attributes: map[string]schema.Attribute{
			"projects": schema.ListNestedAttribute{
				MarkdownDescription: "The list of projects.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the project.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the project.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "A description of the project.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *projectListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data",
			"Expected *client.Client, got an unexpected type. Please report this issue to the provider developers.",
		)
		return
	}
	d.client = c
}

func (d *projectListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	projects, err := d.client.ListProjects(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing projects", fmt.Sprintf("Could not list projects: %s", err))
		return
	}

	var state projectListDataSourceModel
	for _, p := range projects {
		item := projectItemModel{
			UUID: types.StringValue(p.UUID),
			Name: types.StringValue(p.Name),
		}
		item.Description = flex.StringToFramework(p.Description)
		state.Projects = append(state.Projects, item)
	}

	if state.Projects == nil {
		state.Projects = []projectItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

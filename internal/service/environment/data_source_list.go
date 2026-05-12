package environment

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
	_ datasource.DataSource              = (*environmentListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*environmentListDataSource)(nil)
)

// environmentListDataSource is the data source implementation for listing all Coolify environments in a project.
type environmentListDataSource struct {
	client *client.Client
}

// environmentListDataSourceModel maps the data source schema data.
type environmentListDataSourceModel struct {
	ProjectUUID  types.String           `tfsdk:"project_uuid"`
	Environments []environmentItemModel `tfsdk:"environments"`
}

// environmentItemModel maps a single environment in the list.
type environmentItemModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

// NewListDataSource returns a new environments list data source instance.
func NewListDataSource() datasource.DataSource {
	return &environmentListDataSource{}
}

func (d *environmentListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environments"
}

func (d *environmentListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify environments in a project.",
		Attributes: map[string]schema.Attribute{
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"environments": schema.ListNestedAttribute{
				MarkdownDescription: "The list of environments.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The numeric ID of the environment.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the environment.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "A description of the environment.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *environmentListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *environmentListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config environmentListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envs, err := d.client.ListEnvironments(ctx, config.ProjectUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing environments", fmt.Sprintf("Could not list environments: %s", err))
		return
	}

	var state environmentListDataSourceModel
	state.ProjectUUID = config.ProjectUUID
	for _, e := range envs {
		item := environmentItemModel{
			ID:   types.Int64Value(e.ID),
			Name: types.StringValue(e.Name),
		}
		item.Description = flex.StringToFramework(e.Description)
		state.Environments = append(state.Environments, item)
	}

	if state.Environments == nil {
		state.Environments = []environmentItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

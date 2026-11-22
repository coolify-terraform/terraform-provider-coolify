package environment

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	Filters      []filter.Config        `tfsdk:"filter"`
}

// environmentItemModel maps a single environment in the list.
type environmentItemModel struct {
	ID          types.Int64  `tfsdk:"id"`
	UUID        types.String `tfsdk:"uuid"`
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
		MarkdownDescription: "Lists Coolify environments in a project. Each item's `uuid` is the value to pass as `environment_uuid` on applications, databases, and services. `id` is only the numeric database key.",
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
							MarkdownDescription: "The numeric database ID of the environment. This is not the environment UUID.",
							Computed:            true,
						},
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The Coolify environment UUID. Use this as `environment_uuid` on applications, databases, and services so renaming the environment does not replace those resources.",
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
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *environmentListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *environmentListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config environmentListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_environments"})

	envs, err := d.client.ListEnvironments(ctx, config.ProjectUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing environments", fmt.Sprintf("Could not list environments: %s", err))
		return
	}

	envs = filter.Apply(ctx, envs, config.Filters, func(e client.Environment, field string) (string, bool) {
		switch field {
		case "id":
			return filter.Int64ToString(e.ID), true
		case "uuid":
			return e.UUID, true
		case "name":
			return e.Name, true
		case "description":
			return e.Description, true
		default:
			return "", false
		}
	})

	var state environmentListDataSourceModel
	state.ProjectUUID = config.ProjectUUID
	state.Filters = config.Filters
	for _, e := range envs {
		if e.UUID == "" {
			resp.Diagnostics.AddError("Error listing environments", fmt.Sprintf("Coolify returned environment %q in project %s without a uuid.", e.Name, config.ProjectUUID.ValueString()))
			return
		}
		item := environmentItemModel{
			ID:   types.Int64Value(e.ID),
			UUID: types.StringValue(e.UUID),
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

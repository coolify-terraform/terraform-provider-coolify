package environment

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
	_ datasource.DataSource              = (*environmentDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*environmentDataSource)(nil)
)

// environmentDataSource is the data source implementation for a single Coolify environment.
type environmentDataSource struct {
	client *client.Client
}

// environmentDataSourceModel maps the data source schema data.
type environmentDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	UUID        types.String `tfsdk:"uuid"`
	ProjectUUID types.String `tfsdk:"project_uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

// NewDataSource returns a new environment data source instance.
func NewDataSource() datasource.DataSource {
	return &environmentDataSource{}
}

func (d *environmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *environmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads one Coolify environment by project UUID and name. `uuid` is the value to pass as `environment_uuid` on applications, databases, and services. `name` is the lookup key. Coolify also accepts an environment UUID in that path, but this data source stores the name Coolify returns, so keep a name in `name`.",
		Attributes: map[string]schema.Attribute{
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the environment.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the environment.",
				Computed:            true,
			},
			"id": schema.Int64Attribute{
				MarkdownDescription: "The numeric database ID of the environment. This is not the environment UUID.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The Coolify environment UUID. Use this as `environment_uuid` on applications, databases, and services so renaming the environment does not replace those resources.",
				Computed:            true,
			},
		},
	}
}

func (d *environmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *environmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config environmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_environment"})

	projectUUID := config.ProjectUUID.ValueString()
	name := config.Name.ValueString()
	env, err := d.client.GetEnvironment(ctx, projectUUID, name)
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment", fmt.Sprintf("Could not read environment %q in project %s: %s", name, projectUUID, err))
		return
	}

	if env.UUID == "" {
		resp.Diagnostics.AddError("Error reading environment", fmt.Sprintf("Coolify returned environment %q in project %s without a uuid.", name, projectUUID))
		return
	}
	config.ID = types.Int64Value(env.ID)
	config.UUID = types.StringValue(env.UUID)
	config.Name = types.StringValue(env.Name)
	config.Description = types.StringValue(env.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

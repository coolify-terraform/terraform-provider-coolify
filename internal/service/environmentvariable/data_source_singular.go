package environmentvariable

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*envVarDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*envVarDataSource)(nil)
)

type envVarDataSource struct {
	client *client.Client
}

type envVarDataSourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	ApplicationUUID types.String `tfsdk:"application_uuid"`
	ServiceUUID     types.String `tfsdk:"service_uuid"`
	DatabaseUUID    types.String `tfsdk:"database_uuid"`
	Key             types.String `tfsdk:"key"`
	Value           types.String `tfsdk:"value"`
	IsPreview       types.Bool   `tfsdk:"is_preview"`
	IsBuild         types.Bool   `tfsdk:"is_build"`
}

// NewDataSource returns a new singular environment variable data source.
func NewDataSource() datasource.DataSource { return &envVarDataSource{} }

func (d *envVarDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variable"
}

func (d *envVarDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a single environment variable by UUID from a Coolify application, service, or database.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the environment variable.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided.",
				Optional:            true,
				Validators: []validator.String{
					validate.UUID(),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("service_uuid"),
						path.MatchRoot("database_uuid"),
					),
				},
			},
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided.",
				Optional:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the database. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided.",
				Optional:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The variable name.",
				Computed:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The variable value.",
				Computed:            true,
				Sensitive:           true,
			},
			"is_preview": schema.BoolAttribute{
				MarkdownDescription: "Whether available in preview deployments.",
				Computed:            true,
			},
			"is_build": schema.BoolAttribute{
				MarkdownDescription: "Whether available at build time.",
				Computed:            true,
			},
		},
	}
}

func (d *envVarDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *envVarDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config envVarDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_environment_variable"})

	parentType, parentUUID, ok := dsParentTypeAndUUID(config.ApplicationUUID, config.ServiceUUID, config.DatabaseUUID)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	envVars, err := d.client.ListEnvVars(ctx, parentType, parentUUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment variable", fmt.Sprintf("env var %s: %s", config.UUID.ValueString(), err))
		return
	}

	uuid := config.UUID.ValueString()
	ev, found := client.FindEnvVarByUUID(envVars, uuid)
	if !found {
		resp.Diagnostics.AddError("Error reading environment variable",
			fmt.Sprintf("Environment variable with UUID %q not found", uuid))
		return
	}

	config.Key = types.StringValue(ev.Key)
	config.Value = types.StringValue(ev.Value)
	config.IsPreview = types.BoolValue(ev.IsPreview)
	config.IsBuild = types.BoolValue(ev.IsBuild)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

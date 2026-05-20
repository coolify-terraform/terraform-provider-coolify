package environmentvariable

import (
	"context"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*envVarListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*envVarListDataSource)(nil)
)

type envVarListDataSource struct {
	client *client.Client
}

type envVarListModel struct {
	ApplicationUUID      types.String      `tfsdk:"application_uuid"`
	ServiceUUID          types.String      `tfsdk:"service_uuid"`
	DatabaseUUID         types.String      `tfsdk:"database_uuid"`
	EnvironmentVariables []envVarItemModel `tfsdk:"environment_variables"`
	Filters              []filter.Config   `tfsdk:"filter"`
}

type envVarItemModel struct {
	UUID      types.String `tfsdk:"uuid"`
	Key       types.String `tfsdk:"key"`
	Value     types.String `tfsdk:"value"`
	IsPreview types.Bool   `tfsdk:"is_preview"`
	IsBuild   types.Bool   `tfsdk:"is_build"`
}

// dsParentTypeAndUUID resolves which parent UUID attribute is set and returns
// the API parent type ("applications", "services", or "databases") plus UUID.
func dsParentTypeAndUUID(appUUID, svcUUID, dbUUID types.String) (string, string, bool) {
	if !appUUID.IsNull() && !appUUID.IsUnknown() {
		return "applications", appUUID.ValueString(), true
	}
	if !svcUUID.IsNull() && !svcUUID.IsUnknown() {
		return "services", svcUUID.ValueString(), true
	}
	if !dbUUID.IsNull() && !dbUUID.IsUnknown() {
		return "databases", dbUUID.ValueString(), true
	}
	return "", "", false
}

func NewListDataSource() datasource.DataSource { return &envVarListDataSource{} }

func (d *envVarListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variables"
}

func (d *envVarListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all environment variables for a Coolify application, service, or database.",
		Attributes: map[string]schema.Attribute{
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
			"environment_variables": schema.ListNestedAttribute{
				MarkdownDescription: "The list of environment variables.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":       schema.StringAttribute{MarkdownDescription: "The UUID of the environment variable.", Computed: true},
						"key":        schema.StringAttribute{MarkdownDescription: "The variable name.", Computed: true},
						"value":      schema.StringAttribute{MarkdownDescription: "The variable value.", Computed: true, Sensitive: true},
						"is_preview": schema.BoolAttribute{MarkdownDescription: "Whether available in preview deployments.", Computed: true},
						"is_build":   schema.BoolAttribute{MarkdownDescription: "Whether available at build time.", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *envVarListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *envVarListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config envVarListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_environment_variables"})

	parentType, parentUUID, ok := dsParentTypeAndUUID(config.ApplicationUUID, config.ServiceUUID, config.DatabaseUUID)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	envVars, err := d.client.ListEnvVars(ctx, parentType, parentUUID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing environment variables", err.Error())
		return
	}

	envVars = filter.Apply(ctx, envVars, config.Filters, func(ev client.EnvironmentVariable, field string) (string, bool) {
		switch field {
		case "uuid":
			return ev.UUID, true
		case "key":
			return ev.Key, true
		case "value":
			return ev.Value, true
		case "is_preview":
			return filter.BoolToString(ev.IsPreview), true
		case "is_build":
			return filter.BoolToString(ev.IsBuild), true
		default:
			return "", false
		}
	})

	items := make([]envVarItemModel, len(envVars))
	for i, ev := range envVars {
		items[i] = envVarItemModel{
			UUID:      types.StringValue(ev.UUID),
			Key:       types.StringValue(ev.Key),
			Value:     types.StringValue(ev.Value),
			IsPreview: types.BoolValue(ev.IsPreview),
			IsBuild:   types.BoolValue(ev.IsBuild),
		}
	}
	config.EnvironmentVariables = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

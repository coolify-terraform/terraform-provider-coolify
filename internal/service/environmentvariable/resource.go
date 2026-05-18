package environmentvariable

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &environmentVariableResource{}
	_ resource.ResourceWithConfigure      = &environmentVariableResource{}
	_ resource.ResourceWithImportState    = &environmentVariableResource{}
	_ resource.ResourceWithValidateConfig = &environmentVariableResource{}
)

// environmentVariableResource manages a single environment variable on an
// application, service, or database.
type environmentVariableResource struct {
	client *client.Client
}

// environmentVariableResourceModel maps the resource schema to Go types.
type environmentVariableResourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	ApplicationUUID types.String `tfsdk:"application_uuid"`
	ServiceUUID     types.String `tfsdk:"service_uuid"`
	DatabaseUUID    types.String `tfsdk:"database_uuid"`
	Key             types.String `tfsdk:"key"`
	Value           types.String `tfsdk:"value"`
	IsPreview       types.Bool   `tfsdk:"is_preview"`
	IsBuild         types.Bool   `tfsdk:"is_build"`
}

// NewResource returns a new environmentVariableResource instance.
func NewResource() resource.Resource {
	return &environmentVariableResource{}
}

func (r *environmentVariableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variable"
}

func (r *environmentVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an environment variable on a Coolify application, service, or database.\n\n" +
			"~> **Note:** Each instance requires a List API call to read because the Coolify API does not " +
			"provide a singular GET endpoint for environment variables. Large numbers of these resources " +
			"on a single application may cause slower plan/apply times due to this API limitation.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the environment variable.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application to set the variable on. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("service_uuid"),
						path.MatchRoot("database_uuid"),
					),
					validate.UUID(),
				},
			},
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service to set the variable on. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{validate.UUID()},
			},
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the database to set the variable on. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{validate.UUID()},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The name of the environment variable (must be a valid shell variable name). Changing the key forces replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`), "must be a valid environment variable name (letters, digits, underscores; cannot start with a digit)"),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the environment variable.",
				Required:            true,
				Sensitive:           true,
			},
			"is_preview": schema.BoolAttribute{
				MarkdownDescription: "Whether this variable is available in preview deployments. Set it explicitly when you need preview-scoped behavior.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"is_build": schema.BoolAttribute{
				MarkdownDescription: "Whether this variable is available at build time. Supported only for application-scoped environment variables. If omitted during create, Coolify defaults application env vars to `true`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *environmentVariableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *environmentVariableResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config environmentVariableResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.IsBuild.IsNull() || config.IsBuild.IsUnknown() {
		return
	}

	if !config.ApplicationUUID.IsNull() && !config.ApplicationUUID.IsUnknown() {
		return
	}

	if !config.ServiceUUID.IsNull() && !config.ServiceUUID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("is_build"),
			"Unsupported build-time environment variable scope",
			"`is_build` is only supported for application-scoped environment variables because Coolify does not persist `is_buildtime` for services.",
		)
	}

	if !config.DatabaseUUID.IsNull() && !config.DatabaseUUID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("is_build"),
			"Unsupported build-time environment variable scope",
			"`is_build` is only supported for application-scoped environment variables because Coolify does not persist `is_buildtime` for databases.",
		)
	}
}

func (r *environmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentVariableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_environment_variable"})

	isPreview := plan.IsPreview.ValueBool()
	stateIsBuild := false
	var createIsBuild *bool
	if !plan.IsBuild.IsNull() && !plan.IsBuild.IsUnknown() {
		stateIsBuild = plan.IsBuild.ValueBool()
		createIsBuild = &stateIsBuild
	} else if !plan.ApplicationUUID.IsNull() && !plan.ApplicationUUID.IsUnknown() {
		stateIsBuild = true
	}

	ev := client.EnvironmentVariable{
		Key:       plan.Key.ValueString(),
		Value:     plan.Value.ValueString(),
		IsPreview: isPreview,
		IsBuild:   stateIsBuild,
	}

	var createResp *client.CreateEnvVarResponse
	var err error

	//nolint:gocritic // if-else chain dispatches to different client methods; switch not applicable
	if !plan.ApplicationUUID.IsNull() && !plan.ApplicationUUID.IsUnknown() {
		createResp, err = r.client.CreateApplicationEnvVar(ctx, plan.ApplicationUUID.ValueString(), ev, createIsBuild)
	} else if !plan.ServiceUUID.IsNull() && !plan.ServiceUUID.IsUnknown() {
		createResp, err = r.client.CreateServiceEnvVar(ctx, plan.ServiceUUID.ValueString(), ev)
	} else if !plan.DatabaseUUID.IsNull() && !plan.DatabaseUUID.IsUnknown() {
		createResp, err = r.client.CreateDatabaseEnvVar(ctx, plan.DatabaseUUID.ValueString(), ev)
	} else {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error creating environment variable", fmt.Sprintf("env var %s: %s", plan.Key.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(createResp.UUID)
	// Ensure bool fields are known after apply (they may be unknown if
	// the user omitted them and there is no schema default).
	plan.IsPreview = types.BoolValue(isPreview)
	plan.IsBuild = types.BoolValue(stateIsBuild)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": plan.UUID.ValueString()})
}

func (r *environmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state environmentVariableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": state.UUID.ValueString()})

	var envVars []client.EnvironmentVariable
	var err error

	//nolint:gocritic // if-else chain dispatches to different client methods; switch not applicable
	if !state.ApplicationUUID.IsNull() && !state.ApplicationUUID.IsUnknown() {
		envVars, err = r.client.ListApplicationEnvVars(ctx, state.ApplicationUUID.ValueString())
	} else if !state.ServiceUUID.IsNull() && !state.ServiceUUID.IsUnknown() {
		envVars, err = r.client.ListServiceEnvVars(ctx, state.ServiceUUID.ValueString())
	} else if !state.DatabaseUUID.IsNull() && !state.DatabaseUUID.IsUnknown() {
		envVars, err = r.client.ListDatabaseEnvVars(ctx, state.DatabaseUUID.ValueString())
	} else {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading environment variables", fmt.Sprintf("env var %s: %s", state.UUID.ValueString(), err))
		return
	}

	found := false
	for _, ev := range envVars {
		if ev.UUID == state.UUID.ValueString() {
			state.Key = types.StringValue(ev.Key)
			if ev.Value != "" || state.Value.IsNull() || state.Value.IsUnknown() {
				state.Value = types.StringValue(ev.Value)
			}
			state.IsPreview = types.BoolValue(ev.IsPreview)
			state.IsBuild = types.BoolValue(ev.IsBuild)
			found = true
			break
		}
	}

	if !found {
		tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": state.UUID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *environmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan environmentVariableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": plan.UUID.ValueString()})

	ev := client.EnvironmentVariable{
		Key:       plan.Key.ValueString(),
		Value:     plan.Value.ValueString(),
		IsPreview: plan.IsPreview.ValueBool(),
		IsBuild:   plan.IsBuild.ValueBool(),
	}

	var err error

	//nolint:gocritic // if-else chain dispatches to different client methods; switch not applicable
	if !plan.ApplicationUUID.IsNull() && !plan.ApplicationUUID.IsUnknown() {
		err = r.client.UpdateApplicationEnvVar(ctx, plan.ApplicationUUID.ValueString(), ev)
	} else if !plan.ServiceUUID.IsNull() && !plan.ServiceUUID.IsUnknown() {
		err = r.client.UpdateServiceEnvVar(ctx, plan.ServiceUUID.ValueString(), ev)
	} else if !plan.DatabaseUUID.IsNull() && !plan.DatabaseUUID.IsUnknown() {
		err = r.client.UpdateDatabaseEnvVar(ctx, plan.DatabaseUUID.ValueString(), ev)
	} else {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error updating environment variable", fmt.Sprintf("env var %s: %s", plan.UUID.ValueString(), err))
		return
	}

	plan.IsPreview = types.BoolValue(ev.IsPreview)
	plan.IsBuild = types.BoolValue(ev.IsBuild)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *environmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentVariableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": state.UUID.ValueString()})

	var err error

	//nolint:gocritic // if-else chain dispatches to different client methods; switch not applicable
	if !state.ApplicationUUID.IsNull() && !state.ApplicationUUID.IsUnknown() {
		err = r.client.DeleteApplicationEnvVar(ctx, state.ApplicationUUID.ValueString(), state.UUID.ValueString())
	} else if !state.ServiceUUID.IsNull() && !state.ServiceUUID.IsUnknown() {
		err = r.client.DeleteServiceEnvVar(ctx, state.ServiceUUID.ValueString(), state.UUID.ValueString())
	} else if !state.DatabaseUUID.IsNull() && !state.DatabaseUUID.IsUnknown() {
		err = r.client.DeleteDatabaseEnvVar(ctx, state.DatabaseUUID.ValueString(), state.UUID.ValueString())
	} else {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting environment variable", fmt.Sprintf("env var %s: %s", state.UUID.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": state.UUID.ValueString()})
}

func (r *environmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			`Expected "application:{app_uuid}:{env_uuid}", "service:{svc_uuid}:{env_uuid}", or "database:{db_uuid}:{env_uuid}".`,
		)
		return
	}

	resourceType := parts[0]
	parentUUID := parts[1]
	envUUID := parts[2]

	if err := validate.ImportUUID(parentUUID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("parent UUID segment: %s", err))
		return
	}
	if err := validate.ImportUUID(envUUID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("env variable UUID segment: %s", err))
		return
	}

	switch resourceType {
	case "application":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_uuid"), parentUUID)...)
	case "service":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_uuid"), parentUUID)...)
	case "database":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_uuid"), parentUUID)...)
	default:
		resp.Diagnostics.AddError(
			"Invalid import ID type",
			fmt.Sprintf("Expected \"application\", \"service\", or \"database\", got %q.", resourceType),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), envUUID)...)
}

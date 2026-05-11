package environmentvariable

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &environmentVariableResource{}
	_ resource.ResourceWithConfigure   = &environmentVariableResource{}
	_ resource.ResourceWithImportState = &environmentVariableResource{}
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
		MarkdownDescription: "Manages an environment variable on a Coolify application, service, or database.",
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
				MarkdownDescription: "The name of the environment variable (must be a valid shell variable name).",
				Required:            true,
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
				MarkdownDescription: "Whether this variable is available in preview deployments (defaults to `false`).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"is_build": schema.BoolAttribute{
				MarkdownDescription: "Whether this variable is available at build time (defaults to `false`).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *environmentVariableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *environmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentVariableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ev := client.EnvironmentVariable{
		Key:       plan.Key.ValueString(),
		Value:     plan.Value.ValueString(),
		IsPreview: plan.IsPreview.ValueBool(),
		IsBuild:   plan.IsBuild.ValueBool(),
	}

	var createResp *client.CreateEnvVarResponse
	var err error

	//nolint:gocritic // if-else chain dispatches to different client methods; switch not applicable
	if !plan.ApplicationUUID.IsNull() && !plan.ApplicationUUID.IsUnknown() {
		createResp, err = r.client.CreateApplicationEnvVar(ctx, plan.ApplicationUUID.ValueString(), ev)
	} else if !plan.ServiceUUID.IsNull() && !plan.ServiceUUID.IsUnknown() {
		createResp, err = r.client.CreateServiceEnvVar(ctx, plan.ServiceUUID.ValueString(), ev)
	} else if !plan.DatabaseUUID.IsNull() && !plan.DatabaseUUID.IsUnknown() {
		createResp, err = r.client.CreateDatabaseEnvVar(ctx, plan.DatabaseUUID.ValueString(), ev)
	} else {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error creating environment variable", err.Error())
		return
	}

	plan.UUID = types.StringValue(createResp.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *environmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state environmentVariableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading environment variables", err.Error())
		return
	}

	found := false
	for _, ev := range envVars {
		if ev.UUID == state.UUID.ValueString() {
			state.Key = types.StringValue(ev.Key)
			state.Value = types.StringValue(ev.Value)
			state.IsPreview = types.BoolValue(ev.IsPreview)
			state.IsBuild = types.BoolValue(ev.IsBuild)
			found = true
			break
		}
	}

	if !found {
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

	ev := client.EnvironmentVariable{
		UUID:      plan.UUID.ValueString(),
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
		resp.Diagnostics.AddError("Error updating environment variable", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *environmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentVariableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
		resp.Diagnostics.AddError("Error deleting environment variable", err.Error())
		return
	}
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
		resp.Diagnostics.AddError("Invalid Import ID", "parent UUID segment: "+err.Error())
		return
	}
	if err := validate.ImportUUID(envUUID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "env variable UUID segment: "+err.Error())
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

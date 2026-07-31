package environmentvariable

import (
	"context"
	"fmt"
	"regexp"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"

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
	IsRuntime       types.Bool   `tfsdk:"is_runtime"`
	IsLiteral       types.Bool   `tfsdk:"is_literal"`
	IsMultiline     types.Bool   `tfsdk:"is_multiline"`
	IsShownOnce     types.Bool   `tfsdk:"is_shown_once"`
	Comment         types.String `tfsdk:"comment"`
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
			"on a single parent resource may cause slower plan/apply times due to this API limitation.\n\n" +
			"`is_build` and `is_runtime` are **application-only** (Coolify `is_buildtime` / `is_runtime`). " +
			"`is_literal`, `is_multiline`, and `comment` are supported for applications, services, and databases.",
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
				MarkdownDescription: "Whether this variable is available at build time (Coolify `is_buildtime`). Supported only for application-scoped environment variables. If omitted during create, Coolify defaults application env vars to `true`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"is_runtime": schema.BoolAttribute{
				MarkdownDescription: "Whether this variable is available at container runtime (Coolify `is_runtime`). Supported only for application-scoped environment variables. If omitted during create, Coolify defaults application env vars to `true`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"is_literal": schema.BoolAttribute{
				MarkdownDescription: "Whether the value is treated as a literal (no Coolify escaping/expansion). Defaults to `false` when omitted on create.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"is_shown_once": schema.BoolAttribute{
				MarkdownDescription: "Whether the value is revealed only once in the Coolify UI (sensitive-style display). Application env write path only; Coolify default is `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"is_multiline": schema.BoolAttribute{
				MarkdownDescription: "Whether the value is multiline. Defaults to `false` when omitted on create.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"comment": schema.StringAttribute{
				MarkdownDescription: "Optional human-readable comment for the environment variable (max 256 characters on Coolify).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.LengthAtMost(256)},
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

	// Application-only flags: is_build and is_runtime.
	appOnly := []struct {
		attr path.Path
		val  types.Bool
		name string
		api  string
	}{
		{path.Root("is_build"), config.IsBuild, "is_build", "is_buildtime"},
		{path.Root("is_runtime"), config.IsRuntime, "is_runtime", "is_runtime"},
	}

	for _, f := range appOnly {
		if f.val.IsNull() || f.val.IsUnknown() {
			continue
		}
		if !config.ApplicationUUID.IsNull() && !config.ApplicationUUID.IsUnknown() {
			continue
		}
		if !config.ServiceUUID.IsNull() && !config.ServiceUUID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				f.attr,
				"Unsupported environment variable scope",
				fmt.Sprintf("`%s` is only supported for application-scoped environment variables because Coolify does not persist `%s` for services.", f.name, f.api),
			)
		}
		if !config.DatabaseUUID.IsNull() && !config.DatabaseUUID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				f.attr,
				"Unsupported environment variable scope",
				fmt.Sprintf("`%s` is only supported for application-scoped environment variables because Coolify does not persist `%s` for databases.", f.name, f.api),
			)
		}
	}
}

// parentLabel returns a user-friendly singular label for the API slug.
func parentLabel(slug string) string {
	switch slug {
	case "applications":
		return "application"
	case "services":
		return "service"
	case "databases":
		return "database"
	default:
		return slug
	}
}

func parentTypeAndUUID(m *environmentVariableResourceModel) (string, string, bool) {
	if !m.ApplicationUUID.IsNull() && !m.ApplicationUUID.IsUnknown() {
		return "applications", m.ApplicationUUID.ValueString(), true
	}
	if !m.ServiceUUID.IsNull() && !m.ServiceUUID.IsUnknown() {
		return "services", m.ServiceUUID.ValueString(), true
	}
	if !m.DatabaseUUID.IsNull() && !m.DatabaseUUID.IsUnknown() {
		return "databases", m.DatabaseUUID.ValueString(), true
	}
	return "", "", false
}

func boolPtrIfKnown(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func boolPtrKnownOrDefault(v types.Bool, def bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return &def
	}
	b := v.ValueBool()
	return &b
}

func stringPtrIfKnown(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// writeOptsFromPlan builds create/update opts. Application-only flags are
// only set for applications. Shared flags apply to all parents.
// forUpdate must be true on PATCH: Coolify application update uses
// is_literal ?? false and assigns is_multiline without has(), so omitting
// those keys would clear them.
func writeOptsFromPlan(parentType string, plan *environmentVariableResourceModel, forUpdate bool) *client.EnvVarWriteOpts {
	opts := &client.EnvVarWriteOpts{
		Comment: stringPtrIfKnown(plan.Comment),
	}
	if forUpdate && parentType == "applications" {
		opts.IsLiteral = boolPtrKnownOrDefault(plan.IsLiteral, false)
		opts.IsMultiline = boolPtrKnownOrDefault(plan.IsMultiline, false)
		opts.IsShownOnce = boolPtrKnownOrDefault(plan.IsShownOnce, false)
	} else {
		opts.IsLiteral = boolPtrIfKnown(plan.IsLiteral)
		opts.IsMultiline = boolPtrIfKnown(plan.IsMultiline)
		if parentType == "applications" {
			opts.IsShownOnce = boolPtrIfKnown(plan.IsShownOnce)
		}
	}
	if parentType == "applications" {
		opts.IsBuild = boolPtrIfKnown(plan.IsBuild)
		opts.IsRuntime = boolPtrIfKnown(plan.IsRuntime)
	}
	return opts
}

// applyCreateDefaults fills Optional+Computed fields after create when the
// user omitted them (Coolify defaults).
func applyCreateDefaults(parentType string, plan *environmentVariableResourceModel) {
	if plan.IsPreview.IsNull() || plan.IsPreview.IsUnknown() {
		plan.IsPreview = types.BoolValue(false)
	}
	if parentType == "applications" {
		if plan.IsBuild.IsNull() || plan.IsBuild.IsUnknown() {
			plan.IsBuild = types.BoolValue(true)
		}
		if plan.IsRuntime.IsNull() || plan.IsRuntime.IsUnknown() {
			plan.IsRuntime = types.BoolValue(true)
		}
	} else {
		// App-only attributes are not managed for service/database.
		// Coolify model defaults both to true, but the provider surface
		// treats them as false (ValidateConfig rejects user sets).
		plan.IsBuild = types.BoolValue(false)
		plan.IsRuntime = types.BoolValue(false)
	}
	if plan.IsLiteral.IsNull() || plan.IsLiteral.IsUnknown() {
		plan.IsLiteral = types.BoolValue(false)
	}
	if plan.IsMultiline.IsNull() || plan.IsMultiline.IsUnknown() {
		plan.IsMultiline = types.BoolValue(false)
	}
	if plan.IsShownOnce.IsNull() || plan.IsShownOnce.IsUnknown() {
		plan.IsShownOnce = types.BoolValue(false)
	}
	if plan.Comment.IsNull() || plan.Comment.IsUnknown() {
		plan.Comment = types.StringValue("")
	}
}

func flattenEnvToModel(parentType string, ev client.EnvironmentVariable, state *environmentVariableResourceModel) {
	priorValue := ""
	if !state.Value.IsNull() && !state.Value.IsUnknown() {
		priorValue = state.Value.ValueString()
	}
	state.Key = types.StringValue(ev.Key)
	state.Value = types.StringValue(client.PreserveEnvVarValue(ev.Value, priorValue))
	state.IsPreview = types.BoolValue(ev.IsPreview)
	if parentType == "applications" {
		state.IsBuild = types.BoolValue(ev.IsBuild)
		state.IsRuntime = types.BoolValue(ev.IsRuntime)
	} else {
		// Ignore Coolify model defaults (true/true); keep provider semantics.
		state.IsBuild = types.BoolValue(false)
		state.IsRuntime = types.BoolValue(false)
	}
	state.IsLiteral = types.BoolValue(ev.IsLiteral)
	state.IsMultiline = types.BoolValue(ev.IsMultiline)
	state.IsShownOnce = types.BoolValue(ev.IsShownOnce)
	state.Comment = types.StringValue(ev.Comment)
}

func (r *environmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentVariableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_environment_variable"})

	parentType, parentUUID, ok := parentTypeAndUUID(&plan)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	isPreview := false
	if !plan.IsPreview.IsNull() && !plan.IsPreview.IsUnknown() {
		isPreview = plan.IsPreview.ValueBool()
	}

	ev := client.EnvironmentVariable{
		Key:       plan.Key.ValueString(),
		Value:     plan.Value.ValueString(),
		IsPreview: isPreview,
	}
	opts := writeOptsFromPlan(parentType, &plan, false)

	createResp, err := r.client.CreateEnvVar(ctx, parentType, parentUUID, ev, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating environment variable", fmt.Sprintf("env var %s: %s", plan.Key.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(createResp.UUID)
	// Known plan values are preserved; null/unknown get Coolify defaults.
	applyCreateDefaults(parentType, &plan)
	plan.IsPreview = types.BoolValue(isPreview)

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

	parentType, parentUUID, ok := parentTypeAndUUID(&state)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	envVars, err := r.client.ListEnvVars(ctx, parentType, parentUUID)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading environment variables", fmt.Sprintf("%s %s env var %s: %s", parentLabel(parentType), parentUUID, state.UUID.ValueString(), err))
		return
	}

	ev, found := client.FindEnvVarByUUID(envVars, state.UUID.ValueString())
	if !found {
		tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": state.UUID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	flattenEnvToModel(parentType, ev, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *environmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan environmentVariableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": plan.UUID.ValueString()})

	parentType, parentUUID, ok := parentTypeAndUUID(&plan)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	// On update, plan values are known (UseStateForUnknown / user input).
	ev := client.EnvironmentVariable{
		Key:       plan.Key.ValueString(),
		Value:     plan.Value.ValueString(),
		IsPreview: plan.IsPreview.ValueBool(),
	}
	opts := writeOptsFromPlan(parentType, &plan, true)

	if err := r.client.UpdateEnvVar(ctx, parentType, parentUUID, ev, opts); err != nil {
		resp.Diagnostics.AddError("Error updating environment variable", fmt.Sprintf("%s %s env var %s: %s", parentLabel(parentType), parentUUID, plan.UUID.ValueString(), err))
		return
	}

	// Ensure Optional+Computed values are known in state after update.
	applyCreateDefaults(parentType, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *environmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentVariableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": state.UUID.ValueString()})

	parentType, parentUUID, ok := parentTypeAndUUID(&state)
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	if err := r.client.DeleteEnvVar(ctx, parentType, parentUUID, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting environment variable", fmt.Sprintf("%s %s env var %s: %s", parentLabel(parentType), parentUUID, state.UUID.ValueString(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_environment_variable", "uuid": state.UUID.ValueString()})
}

func (r *environmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	validate.ImportParentChild(ctx, req, resp,
		[]string{"application", "service", "database"}, "env variable")
}

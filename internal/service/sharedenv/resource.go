package sharedenv

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*sharedEnvResource)(nil)
	_ resource.ResourceWithConfigure   = (*sharedEnvResource)(nil)
	_ resource.ResourceWithImportState = (*sharedEnvResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*sharedEnvResource)(nil)
)

type sharedEnvResource struct{ client *client.Client }

type sharedEnvModel struct {
	ID          types.String `tfsdk:"id"`
	EnvID       types.String `tfsdk:"env_id"`
	Scope       types.String `tfsdk:"scope"`
	ProjectUUID types.String `tfsdk:"project_uuid"`
	Environment types.String `tfsdk:"environment"`
	ServerUUID  types.String `tfsdk:"server_uuid"`
	Key         types.String `tfsdk:"key"`
	Value       types.String `tfsdk:"value"`
	IsLiteral   types.Bool   `tfsdk:"is_literal"`
	IsMultiline types.Bool   `tfsdk:"is_multiline"`
	IsShownOnce types.Bool   `tfsdk:"is_shown_once"`
	Comment     types.String `tfsdk:"comment"`
}

func NewResource() resource.Resource { return &sharedEnvResource{} }

func (r *sharedEnvResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shared_environment_variable"
}

func (r *sharedEnvResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify shared environment variable at team, project, environment, or server scope. Requires Coolify >= v4.3.0.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true, MarkdownDescription: "Composite import id.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"env_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Coolify env id.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"scope": schema.StringAttribute{Required: true, MarkdownDescription: "One of `team`, `project`, `environment`, or `server`. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf("team", "project", "environment", "server")}},
			"project_uuid": schema.StringAttribute{Optional: true, MarkdownDescription: "Required for project and environment scopes.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"environment":  schema.StringAttribute{Optional: true, MarkdownDescription: "Environment name or UUID. Required for environment scope.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"server_uuid":  schema.StringAttribute{Optional: true, MarkdownDescription: "Required for server scope.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Variable name. Letters, digits, underscores, and dots; must start with a letter or underscore. Hyphens are rejected by Coolify. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`), "must be a valid environment variable name (letters, digits, underscores, dots; cannot start with a digit; hyphens are not allowed)"),
				},
			},
			"value": schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Variable value. Preserved when GET omits it."},
			"is_literal": schema.BoolAttribute{
				Optional: true, Computed: true,
				MarkdownDescription: "When true, Coolify stores the value without expanding variables. Omit to keep the stored value.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"is_multiline": schema.BoolAttribute{
				Optional: true, Computed: true,
				MarkdownDescription: "When true, the value is multiline. Omit to keep the stored value.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"is_shown_once": schema.BoolAttribute{
				Optional: true, Computed: true,
				MarkdownDescription: "When true, Coolify reveals the value only once. Omit to keep the stored value. A saved `true` stays locked: the provider rejects a later `false`, because Coolify would keep `true` and the plan would never settle.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"comment": schema.StringAttribute{
				Optional: true, Computed: true,
				MarkdownDescription: "Note stored with the variable. Omit to keep the stored note. Set an empty string to clear it.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *sharedEnvResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.Config.Raw.IsNull() {
		return
	}
	var state, config sharedEnvModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.IsShownOnce.IsNull() || state.IsShownOnce.IsUnknown() || !state.IsShownOnce.ValueBool() {
		return
	}
	if config.IsShownOnce.IsNull() || config.IsShownOnce.IsUnknown() || config.IsShownOnce.ValueBool() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		path.Root("is_shown_once"),
		"Environment variable stays locked",
		"Coolify keeps `is_shown_once` true once it is saved, so a later false does not reveal the value. Remove `is_shown_once = false`, or omit the attribute.",
	)
}

func (r *sharedEnvResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func sharedEnvInput(plan sharedEnvModel, includeKey bool) client.SharedEnvInput {
	input := client.SharedEnvInput{Value: plan.Value.ValueString(), Comment: sharedCommentPtr(plan.Comment)}
	if includeKey {
		input.Key = plan.Key.ValueString()
	}
	if !plan.IsLiteral.IsNull() && !plan.IsLiteral.IsUnknown() {
		v := plan.IsLiteral.ValueBool()
		input.IsLiteral = &v
	}
	if !plan.IsMultiline.IsNull() && !plan.IsMultiline.IsUnknown() {
		v := plan.IsMultiline.ValueBool()
		input.IsMultiline = &v
	}
	if !plan.IsShownOnce.IsNull() && !plan.IsShownOnce.IsUnknown() {
		v := plan.IsShownOnce.ValueBool()
		input.IsShownOnce = &v
	}
	return input
}

func sharedCommentPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// sharedCommentFromAPI keeps an omitted comment null when Coolify has none,
// and keeps a configured empty string so a clear survives refresh.
func sharedCommentFromAPI(current types.String, api string) types.String {
	if api != "" {
		return types.StringValue(api)
	}
	if current.IsNull() || current.IsUnknown() {
		return types.StringNull()
	}
	return types.StringValue("")
}

func (m sharedEnvModel) scopeArgs() (scope, project, env, server string) {
	return m.Scope.ValueString(), m.ProjectUUID.ValueString(), m.Environment.ValueString(), m.ServerUUID.ValueString()
}

func sharedEnvImportID(scope, project, env, server, envID string) string {
	switch scope {
	case "team":
		return "team:" + envID
	case "project":
		return "project:" + project + ":" + envID
	case "environment":
		return "environment:" + project + ":" + env + ":" + envID
	case "server":
		return "server:" + server + ":" + envID
	default:
		return strings.Join([]string{scope, project, env, server, envID}, ":")
	}
}

func (r *sharedEnvResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sharedEnvModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	scope, project, env, server := plan.scopeArgs()
	input := sharedEnvInput(plan, true)
	created, err := r.client.CreateSharedEnv(ctx, scope, project, env, server, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating shared environment variable", err.Error())
		return
	}
	plan.EnvID = types.StringValue(strconv.FormatInt(created.ID, 10))
	if created.UUID != "" {
		plan.EnvID = types.StringValue(created.UUID)
	}
	if created.ID != 0 {
		plan.EnvID = types.StringValue(strconv.FormatInt(created.ID, 10))
	}
	plan.ID = types.StringValue(sharedEnvImportID(scope, project, env, server, plan.EnvID.ValueString()))
	if created.Value != "" {
		plan.Value = types.StringValue(created.Value)
	}
	if created.ID == 0 && created.UUID == "" {
		// Create response is often {id} only; keep planned flags.
	} else {
		if input.IsLiteral == nil {
			plan.IsLiteral = types.BoolValue(created.IsLiteral)
		}
		if input.IsMultiline == nil {
			plan.IsMultiline = types.BoolValue(created.IsMultiline)
		}
		if input.IsShownOnce == nil {
			plan.IsShownOnce = types.BoolValue(created.IsShownOnce)
		}
	}
	if plan.Comment.IsNull() || plan.Comment.IsUnknown() {
		plan.Comment = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sharedEnvResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sharedEnvModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	scope, project, env, server := state.scopeArgs()
	list, err := r.client.ListSharedEnvs(ctx, scope, project, env, server)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading shared environment variables", err.Error())
		return
	}
	found := false
	for _, ev := range list {
		id := strconv.FormatInt(ev.ID, 10)
		if id == state.EnvID.ValueString() || ev.UUID == state.EnvID.ValueString() || ev.Key == state.Key.ValueString() {
			state.EnvID = types.StringValue(id)
			state.Key = types.StringValue(ev.Key)
			if ev.Value != "" {
				state.Value = types.StringValue(ev.Value)
			}
			state.IsLiteral = types.BoolValue(ev.IsLiteral)
			state.IsMultiline = types.BoolValue(ev.IsMultiline)
			state.IsShownOnce = types.BoolValue(ev.IsShownOnce)
			state.Comment = sharedCommentFromAPI(state.Comment, ev.Comment)
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

func (r *sharedEnvResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan sharedEnvModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	scope, project, env, server := plan.scopeArgs()
	input := sharedEnvInput(plan, false)
	_, err := r.client.UpdateSharedEnv(ctx, scope, project, env, server, plan.EnvID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating shared environment variable", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sharedEnvResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state sharedEnvModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	scope, project, env, server := state.scopeArgs()
	if err := r.client.DeleteSharedEnv(ctx, scope, project, env, server, state.EnvID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting shared environment variable", err.Error())
	}
}

func (r *sharedEnvResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) < 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected scope:...:env_id (team:env_id, project:project_uuid:env_id, environment:project_uuid:env:env_id, server:server_uuid:env_id)")
		return
	}
	scope := parts[0]
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope"), scope)...)
	switch scope {
	case "team":
		if len(parts) != 2 {
			resp.Diagnostics.AddError("Invalid import ID", "Expected team:env_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("env_id"), parts[1])...)
	case "project":
		if len(parts) != 3 {
			resp.Diagnostics.AddError("Invalid import ID", "Expected project:project_uuid:env_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_uuid"), parts[1])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("env_id"), parts[2])...)
	case "environment":
		if len(parts) != 4 {
			resp.Diagnostics.AddError("Invalid import ID", "Expected environment:project_uuid:environment:env_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_uuid"), parts[1])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), parts[2])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("env_id"), parts[3])...)
	case "server":
		if len(parts) != 3 {
			resp.Diagnostics.AddError("Invalid import ID", "Expected server:server_uuid:env_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_uuid"), parts[1])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("env_id"), parts[2])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("unknown scope %q", scope))
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

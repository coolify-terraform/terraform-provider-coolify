package envsbulk

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*envsBulkResource)(nil)
	_ resource.ResourceWithConfigure   = (*envsBulkResource)(nil)
	_ resource.ResourceWithImportState = (*envsBulkResource)(nil)
)

type envsBulkResource struct {
	client *client.Client
}

type envsBulkModel struct {
	ResourceType types.String `tfsdk:"resource_type"`
	ResourceUUID types.String `tfsdk:"resource_uuid"`
	Variables    types.Map    `tfsdk:"variables"`
}

func NewResource() resource.Resource {
	return &envsBulkResource{}
}

func (r *envsBulkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_envs_bulk"
}

func (r *envsBulkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages environment variables for a Coolify resource as one set. The variables in the map are sent in a single API call.\n\n~> **Note:** Destroying this resource removes it from Terraform state but **does not** delete the environment variables from Coolify. Removing a key from `variables` does delete that variable in Coolify. Variables that were never in this resource stay in place. After importing, keep every variable you still want in the map.",
		Attributes: map[string]schema.Attribute{
			"resource_type": schema.StringAttribute{
				MarkdownDescription: "The type of the target resource. Must be one of: `application`, `database`, `service`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("application", "database", "service"),
				},
			},
			"resource_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the target resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"variables": schema.MapAttribute{
				MarkdownDescription: "A map of environment variable key-value pairs.",
				Required:            true,
				ElementType:         types.StringType,
				Sensitive:           true,
			},
		},
	}
}

func (r *envsBulkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *envsBulkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan envsBulkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{
		"resource_type":        "coolify_envs_bulk",
		"parent_resource_type": plan.ResourceType.ValueString(),
		"parent_resource_uuid": plan.ResourceUUID.ValueString(),
	})

	if err := r.bulkUpdate(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error creating bulk env vars",
			fmt.Sprintf("%s %s: %s", plan.ResourceType.ValueString(), plan.ResourceUUID.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *envsBulkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state envsBulkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resType := state.ResourceType.ValueString()
	uuid := state.ResourceUUID.ValueString()

	tflog.Debug(ctx, "reading resource", map[string]interface{}{
		"resource_type":        "coolify_envs_bulk",
		"parent_resource_type": resType,
		"parent_resource_uuid": uuid,
	})

	envs, err := r.listEnvVars(ctx, resType, uuid)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading bulk env vars",
			fmt.Sprintf("%s %s: %s", resType, uuid, err),
		)
		return
	}

	vars := flattenEnvVars(envs)

	// Only keep keys that are in the current state to avoid pulling in
	// environment variables managed by other resources.
	stateVars := make(map[string]string)
	diags := state.Variables.ElementsAs(ctx, &stateVars, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	managed := make(map[string]string, len(stateVars))
	for k, prior := range stateVars {
		if v, ok := vars[k]; ok {
			managed[k] = client.PreserveEnvVarValue(v, prior)
		}
	}

	mapVal, diags := types.MapValueFrom(ctx, types.StringType, managed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Variables = mapVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *envsBulkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior envsBulkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{
		"resource_type":        "coolify_envs_bulk",
		"parent_resource_type": plan.ResourceType.ValueString(),
		"parent_resource_uuid": plan.ResourceUUID.ValueString(),
	})

	if err := r.deleteDroppedKeys(ctx, &prior, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting removed environment variables",
			fmt.Sprintf("%s %s: %s", plan.ResourceType.ValueString(), plan.ResourceUUID.ValueString(), err),
		)
		return
	}

	if err := r.bulkUpdate(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error updating bulk env vars",
			fmt.Sprintf("%s %s: %s", plan.ResourceType.ValueString(), plan.ResourceUUID.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *envsBulkResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: removing this resource from state stops managing the variables
	// but leaves them in place on the Coolify resource. Sending an empty
	// bulk update would wipe ALL environment variables (including ones not
	// managed by this resource), which is destructive and undesirable.
	tflog.Debug(ctx, "deleting resource (no-op, variables left in place)", map[string]interface{}{"resource_type": "coolify_envs_bulk"})
}

func (r *envsBulkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: resource_type:resource_uuid (colon separator, consistent with all other resources).
	// Also accepts resource_type/resource_uuid for backward compatibility.
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		parts = strings.SplitN(req.ID, "/", 2)
	}
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Import ID", "Expected format: resource_type:resource_uuid")
		return
	}

	resType := parts[0]
	uuid := parts[1]

	switch resType {
	case "application", "database", "service":
	default:
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("resource_type %q must be one of: application, database, service", resType))
		return
	}
	if err := validate.ImportUUID(uuid); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("resource UUID segment: %s", err))
		return
	}

	envs, err := r.listEnvVars(ctx, resType, uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing bulk env vars",
			fmt.Sprintf("%s %s: %s", resType, uuid, err),
		)
		return
	}

	vars := flattenEnvVars(envs)

	mapVal, diags := types.MapValueFrom(ctx, types.StringType, vars)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := envsBulkModel{
		ResourceType: types.StringValue(resType),
		ResourceUUID: types.StringValue(uuid),
		Variables:    mapVal,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func flattenEnvVars(envs []client.EnvironmentVariable) map[string]string {
	collapsed := client.PreferNonPreviewEnvVarsByKey(envs)
	vars := make(map[string]string, len(collapsed))
	for key, ev := range collapsed {
		vars[key] = ev.Value
	}
	return vars
}

// deleteDroppedKeys removes Coolify env vars whose keys left the map.
// Keys that were never in this resource are not listed in prior state, so
// they stay. Both preview and non-preview rows for a dropped key are deleted.
func (r *envsBulkResource) deleteDroppedKeys(ctx context.Context, prior, plan *envsBulkModel) error {
	priorVars := map[string]string{}
	planVars := map[string]string{}
	if diags := prior.Variables.ElementsAs(ctx, &priorVars, false); diags.HasError() {
		return fmt.Errorf("reading prior variables: %s", diags.Errors())
	}
	if diags := plan.Variables.ElementsAs(ctx, &planVars, false); diags.HasError() {
		return fmt.Errorf("reading planned variables: %s", diags.Errors())
	}
	var dropped []string
	for key := range priorVars {
		if _, ok := planVars[key]; !ok {
			dropped = append(dropped, key)
		}
	}
	if len(dropped) == 0 {
		return nil
	}
	sort.Strings(dropped)
	envs, err := r.listEnvVars(ctx, prior.ResourceType.ValueString(), prior.ResourceUUID.ValueString())
	if err != nil {
		return err
	}
	drop := make(map[string]struct{}, len(dropped))
	for _, key := range dropped {
		drop[key] = struct{}{}
	}
	parent := apiParentType(prior.ResourceType.ValueString())
	parentUUID := prior.ResourceUUID.ValueString()
	for _, ev := range envs {
		if _, ok := drop[ev.Key]; !ok {
			continue
		}
		if ev.UUID == "" {
			return fmt.Errorf("environment variable %s has no uuid", ev.Key)
		}
		if err := r.client.DeleteEnvVar(ctx, parent, parentUUID, ev.UUID); err != nil && !client.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *envsBulkResource) bulkUpdate(ctx context.Context, model *envsBulkModel) error {
	vars := make(map[string]string)
	diags := model.Variables.ElementsAs(ctx, &vars, false)
	if diags.HasError() {
		return fmt.Errorf("reading variables map: %s", diags.Errors())
	}

	entries := make([]client.EnvVarEntry, 0, len(vars))
	// Sort keys for deterministic ordering.
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		entries = append(entries, client.EnvVarEntry{Key: k, Value: vars[k]})
	}

	input := client.BulkEnvVarInput{Variables: entries}
	return r.doBulk(ctx, model.ResourceType.ValueString(), model.ResourceUUID.ValueString(), input)
}

// apiParentType maps the schema's singular resource_type to the API's plural path segment.
func apiParentType(resType string) string { return resType + "s" }

func (r *envsBulkResource) doBulk(ctx context.Context, resType, uuid string, input client.BulkEnvVarInput) error {
	return r.client.BulkUpdateEnvVars(ctx, apiParentType(resType), uuid, input)
}

func (r *envsBulkResource) listEnvVars(ctx context.Context, resType, uuid string) ([]client.EnvironmentVariable, error) {
	return r.client.ListEnvVars(ctx, apiParentType(resType), uuid)
}

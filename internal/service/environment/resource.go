package environment

import (
	"context"
	"fmt"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*environmentResource)(nil)
	_ resource.ResourceWithConfigure   = (*environmentResource)(nil)
	_ resource.ResourceWithImportState = (*environmentResource)(nil)
)

// environmentResource is the resource implementation for a Coolify environment.
type environmentResource struct {
	client *client.Client
}

// environmentResourceModel maps the resource schema data.
type environmentResourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	ProjectUUID types.String `tfsdk:"project_uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

// NewResource returns a new environment resource instance.
func NewResource() resource.Resource {
	return &environmentResource{}
}

func (r *environmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *environmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify environment within a project. Every project auto-creates a `production` environment. Use this resource to create additional environments (e.g., `staging`, `dev`). Applications, databases, and services reference environments via their `environment_name` attribute.\n\n~> **Warning:** Deleting an environment will cascade-delete all applications, databases, and services within it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The numeric ID of the environment.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project this environment belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the environment. Coolify accepts in-place rename via PATCH.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the environment. Sent on PATCH after create (Coolify create rejects description) and on subsequent updates.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *environmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *environmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_environment"})

	input := client.CreateEnvironmentInput{
		Name: plan.Name.ValueString(),
	}

	_, err := r.client.CreateEnvironment(ctx, plan.ProjectUUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating environment",
			fmt.Sprintf("project %s: %s", plan.ProjectUUID.ValueString(), err))
		return
	}

	if plan.ID.IsUnknown() {
		plan.ID = types.Int64Null()
	}
	if plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
	}

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID := plan.ProjectUUID.ValueString()
	name := plan.Name.ValueString()

	// Read back the full environment to populate computed fields.
	diags := r.readEnvironment(ctx, projectUUID, name, &plan)
	if diags.HasError() {
		resp.Diagnostics.AddError(
			"Environment created but refresh failed",
			fmt.Sprintf("Coolify created environment %s/%s, but the provider could not read it back: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", projectUUID, name, diags.Errors()[0].Detail()),
		)
		return
	}
	resp.Diagnostics.Append(diags...)

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() && plan.Description.ValueString() != "" {
		desc := plan.Description.ValueString()
		updated, err := r.client.UpdateEnvironment(ctx, projectUUID, name, client.UpdateEnvironmentInput{Description: &desc})
		if err != nil {
			resp.Diagnostics.AddError("Error setting environment description",
				fmt.Sprintf("environment %s/%s was created but PATCH description failed: %s", projectUUID, name, err))
			return
		}
		if updated.Description != "" {
			plan.Description = types.StringValue(updated.Description)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_environment", "name": plan.Name.ValueString()})
}

func (r *environmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state environmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID := state.ProjectUUID.ValueString()
	name := state.Name.ValueString()

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_environment", "name": name})

	env, err := r.client.GetEnvironment(ctx, projectUUID, name)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_environment", "name": name})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading environment", fmt.Sprintf("Could not read environment %s/%s: %s", projectUUID, name, err))
		return
	}

	state.ID = types.Int64Value(env.ID)
	state.Name = types.StringValue(env.Name)
	if env.Description != "" {
		state.Description = types.StringValue(env.Description)
	} else if state.Description.IsNull() || state.Description.IsUnknown() {
		state.Description = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_environment", "name": plan.Name.ValueString()})

	input := client.UpdateEnvironmentInput{}
	name := plan.Name.ValueString()
	desc := ""
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc = plan.Description.ValueString()
	}
	input.Name = &name
	input.Description = &desc

	pathName := state.Name.ValueString()
	updated, err := r.client.UpdateEnvironment(ctx, plan.ProjectUUID.ValueString(), pathName, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating environment",
			fmt.Sprintf("environment %s/%s: %s", plan.ProjectUUID.ValueString(), pathName, err))
		return
	}
	if updated.ID != 0 {
		plan.ID = types.Int64Value(updated.ID)
	} else {
		plan.ID = state.ID
	}
	if updated.Name != "" {
		plan.Name = types.StringValue(updated.Name)
	}
	if updated.Description != "" {
		plan.Description = types.StringValue(updated.Description)
	} else if plan.Description.IsNull() || plan.Description.IsUnknown() {
		plan.Description = types.StringValue("")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *environmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID := state.ProjectUUID.ValueString()
	name := state.Name.ValueString()

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_environment", "name": name})

	err := r.client.DeleteEnvironment(ctx, projectUUID, name)
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting environment", fmt.Sprintf("Could not delete environment %s in project %s: %s", name, projectUUID, err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_environment", "name": name})
}

func (r *environmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			`Expected "project_uuid:environment_name".`,
		)
		return
	}

	if err := validate.ImportUUID(parts[0]); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("project UUID segment: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_uuid"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

// readEnvironment fetches the environment from the API and updates the model in place.
func (r *environmentResource) readEnvironment(ctx context.Context, projectUUID, name string, model *environmentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	env, err := r.client.GetEnvironment(ctx, projectUUID, name)
	if err != nil {
		diags.AddError("Error reading environment", fmt.Sprintf("Could not read environment %s/%s after create: %s", projectUUID, name, err))
		return diags
	}

	model.ID = types.Int64Value(env.ID)
	model.Name = types.StringValue(env.Name)
	if env.Description != "" {
		model.Description = types.StringValue(env.Description)
	} else if model.Description.IsUnknown() {
		model.Description = types.StringValue("")
	}

	return diags
}

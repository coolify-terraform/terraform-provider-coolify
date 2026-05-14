package environment

import (
	"context"
	"fmt"
	"strings"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
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
		MarkdownDescription: "Manages a Coolify environment within a project. Every project auto-creates a `production` environment. Use this resource to create additional environments (e.g. `staging`, `dev`). Applications, databases, and services reference environments via their `environment_name` attribute.",
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
				MarkdownDescription: "The name of the environment. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the environment. Note: the Coolify API does not support updating this field after creation; changes are stored in Terraform state only.",
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
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data",
			"Expected *client.Client, got an unexpected type. Please report this issue to the provider developers.",
		)
		return
	}
	r.client = c
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
		resp.Diagnostics.AddError("Error creating environment", err.Error())
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading environment", fmt.Sprintf("Could not read environment %s/%s: %s", projectUUID, name, err))
		return
	}

	state.ID = types.Int64Value(env.ID)
	state.Name = types.StringValue(env.Name)
	// Coolify does not store or return description via the API.
	// Preserve state value; resolve null (e.g. after import) to empty.
	if state.Description.IsNull() || state.Description.IsUnknown() {
		state.Description = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_environment", "name": plan.Name.ValueString()})

	// The Coolify API has no PATCH endpoint for environments, so we persist
	// the updated description to state without an API call.
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
		resp.Diagnostics.AddError("Error deleting environment", err.Error())
	}
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
		resp.Diagnostics.AddError("Invalid Import ID", "project UUID segment: "+err.Error())
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
	// Coolify does not store or return description via the API.
	// Keep the plan/state value; only resolve Unknown to empty.
	if model.Description.IsUnknown() {
		model.Description = types.StringValue("")
	}

	return diags
}

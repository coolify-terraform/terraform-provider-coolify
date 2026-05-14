package project

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*projectResource)(nil)
	_ resource.ResourceWithImportState = (*projectResource)(nil)
	_ resource.ResourceWithConfigure   = (*projectResource)(nil)
)

// projectResource is the resource implementation for a Coolify project.
type projectResource struct {
	client *client.Client
}

// projectResourceModel maps the resource schema data.
type projectResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

// NewResource returns a new project resource instance.
func NewResource() resource.Resource {
	return &projectResource{}
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify project.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the project.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the project.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_project"})

	input := client.CreateProjectInput{
		Name: plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		input.Description = plan.Description.ValueString()
	}

	project, err := r.client.CreateProject(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating project", err.Error())
		return
	}

	plan.UUID = types.StringValue(project.UUID)
	if plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
	}

	// Save partial state immediately so the resource is tracked even if
	// the read-back fails. Without this, a transient error after create
	// leaves an orphaned resource in Coolify with no Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back the full project to populate all fields.
	diags := r.readProject(ctx, project.UUID, &plan)
	if diags.HasError() {
		resp.Diagnostics.AddError(
			"Project created but refresh failed",
			fmt.Sprintf("Coolify created project %s, but the provider could not read it back: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", project.UUID, diags.Errors()[0].Detail()),
		)
		return
	}
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_project", "uuid": state.UUID.ValueString()})

	project, err := r.client.GetProject(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// The project was deleted outside of Terraform; remove from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project", fmt.Sprintf("Could not read project %s: %s", state.UUID.ValueString(), err))
		return
	}

	state.UUID = types.StringValue(project.UUID)
	state.Name = types.StringValue(project.Name)
	state.Description = flex.StringToFramework(project.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_project", "uuid": state.UUID.ValueString()})

	input := client.UpdateProjectInput{
		Name:        flex.StringIfChanged(plan.Name, state.Name),
		Description: flex.StringIfChanged(plan.Description, state.Description),
	}

	project, err := r.client.UpdateProject(ctx, state.UUID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating project", fmt.Sprintf("Could not update project %s: %s", state.UUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(project.UUID)
	plan.Name = types.StringValue(project.Name)
	plan.Description = flex.StringToFramework(project.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_project", "uuid": state.UUID.ValueString()})

	// Coolify deletes applications and databases asynchronously. When
	// terraform destroy runs, child resources are deleted first but
	// Coolify may not have finished removing them by the time the project
	// delete is attempted. Retry for up to 60 seconds.
	uuid := state.UUID.ValueString()
	var err error
	attempt := 0
retryLoop:
	for range 12 {
		attempt++
		err = r.client.DeleteProject(ctx, uuid)
		if err == nil {
			return
		}
		if client.IsNotFound(err) {
			return
		}
		if !strings.Contains(err.Error(), "has resources") {
			break
		}
		tflog.Debug(ctx, "retrying project delete", map[string]interface{}{"attempt": attempt, "uuid": uuid})
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break retryLoop
		case <-time.After(5 * time.Second):
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting project", err.Error())
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// readProject fetches the project from the API and updates the model in place.
func (r *projectResource) readProject(ctx context.Context, uuid string, model *projectResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	project, err := r.client.GetProject(ctx, uuid)
	if err != nil {
		diags.AddError("Error reading project", fmt.Sprintf("Could not read project %s: %s", uuid, err))
		return diags
	}

	model.UUID = types.StringValue(project.UUID)
	model.Name = types.StringValue(project.Name)
	model.Description = flex.StringToFramework(project.Description)

	return diags
}

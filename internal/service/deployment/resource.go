package deployment

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*deploymentResource)(nil)
	_ resource.ResourceWithConfigure   = (*deploymentResource)(nil)
	_ resource.ResourceWithImportState = (*deploymentResource)(nil)
)

// deploymentResource triggers a deployment for a Coolify application.
type deploymentResource struct {
	client *client.Client
}

// deploymentResourceModel maps the resource schema data.
type deploymentResourceModel struct {
	ApplicationUUID types.String `tfsdk:"application_uuid"`
	UUID            types.String `tfsdk:"uuid"`
	Status          types.String `tfsdk:"status"`
	Triggers        types.Map    `tfsdk:"triggers"`
}

// NewResource returns a new deployment resource instance.
func NewResource() resource.Resource {
	return &deploymentResource{}
}

func (r *deploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *deploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Triggers a deployment for a Coolify application. Changing the triggers map forces a new deployment.",
		Attributes: map[string]schema.Attribute{
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application to deploy.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{validate.UUID()},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the deployment.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the deployment.",
				Computed:            true,
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "An arbitrary map of values that, when changed, triggers a new deployment.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *deploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *deploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appUUID := plan.ApplicationUUID.ValueString()
	result, err := r.client.RestartApplication(ctx, appUUID)
	if err != nil {
		resp.Diagnostics.AddError("Error Triggering Deployment", fmt.Sprintf("Could not restart application %s: %s", appUUID, err))
		return
	}

	plan.UUID = types.StringValue(result.DeploymentUUID)

	// Read back the deployment status.
	dep, err := r.client.GetDeployment(ctx, result.DeploymentUUID)
	if err != nil {
		resp.Diagnostics.AddWarning("Could not read deployment status", fmt.Sprintf("Deployment was triggered but status could not be read: %s. Defaulting to 'queued'.", err))
		plan.Status = types.StringValue("queued")
	} else {
		plan.Status = types.StringValue(dep.Status)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dep, err := r.client.GetDeployment(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Deployment", fmt.Sprintf("Could not read deployment %s: %s", state.UUID.ValueString(), err))
		return
	}

	state.Status = types.StringValue(dep.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *deploymentResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable attributes use RequiresReplace, so Update is never called.
	resp.Diagnostics.AddError("Update not supported", "Deployment resources are replaced, not updated.")
}

func (r *deploymentResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Deployments cannot be undone; delete is a no-op.
}

func (r *deploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

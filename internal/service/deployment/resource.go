package deployment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
	ApplicationUUID   types.String   `tfsdk:"application_uuid"`
	UUID              types.String   `tfsdk:"uuid"`
	Status            types.String   `tfsdk:"status"`
	WaitForCompletion types.Bool     `tfsdk:"wait_for_completion"`
	Triggers          types.Map      `tfsdk:"triggers"`
}

// NewResource returns a new deployment resource instance.
func NewResource() resource.Resource {
	return &deploymentResource{}
}

func (r *deploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *deploymentResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Triggers a deployment for a Coolify application. Changing the triggers map forces a new deployment.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
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
				MarkdownDescription: "The current status of the deployment. Possible values: `queued`, `in_progress`, `finished`, `error`. The deployment may still be `in_progress` when `terraform apply` completes.",
				Computed:            true,
			},
			"wait_for_completion": schema.BoolAttribute{
				MarkdownDescription: "When `true`, the resource waits until the deployment reaches `finished` or `error` status before completing. On `error`, the apply fails with a diagnostic. Default `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
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
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
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

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_deployment"})

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	appUUID := plan.ApplicationUUID.ValueString()
	result, err := r.client.RestartApplication(ctx, appUUID)
	if err != nil {
		resp.Diagnostics.AddError("Error triggering deployment", fmt.Sprintf("Could not restart application %s: %s", appUUID, err))
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

	// Save partial state so the deployment is tracked.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.WaitForCompletion.ValueBool() {
		r.pollDeployment(ctx, result.DeploymentUUID, &plan, resp)
	}
}

func (r *deploymentResource) pollDeployment(ctx context.Context, uuid string, plan *deploymentResourceModel, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "waiting for deployment completion", map[string]interface{}{"uuid": uuid})
	for {
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Deployment timed out", fmt.Sprintf("Deployment %s did not complete within the configured timeout. Last status: %s", uuid, plan.Status.ValueString()))
			return
		case <-time.After(5 * time.Second):
		}
		dep, err := r.client.GetDeployment(ctx, uuid)
		if err != nil {
			resp.Diagnostics.AddError("Error polling deployment", fmt.Sprintf("Could not read deployment %s: %s", uuid, err))
			return
		}
		plan.Status = types.StringValue(dep.Status)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		if dep.Status == "finished" || dep.Status == "error" {
			if dep.Status == "error" {
				resp.Diagnostics.AddError("Deployment failed", fmt.Sprintf("Deployment %s finished with status 'error'", uuid))
			}
			return
		}
	}
}

func (r *deploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_deployment", "uuid": state.UUID.ValueString()})

	dep, err := r.client.GetDeployment(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading deployment", fmt.Sprintf("Could not read deployment %s: %s", state.UUID.ValueString(), err))
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
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			`Expected "application_uuid:deployment_uuid".`,
		)
		return
	}
	if err := validate.ImportUUID(parts[0]); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "application UUID segment: "+err.Error())
		return
	}
	if err := validate.ImportUUID(parts[1]); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "deployment UUID segment: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_uuid"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[1])...)
}

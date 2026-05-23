package applicationpreview

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
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
	_ resource.Resource              = (*applicationPreviewResource)(nil)
	_ resource.ResourceWithConfigure = (*applicationPreviewResource)(nil)
)

type applicationPreviewResource struct {
	client *client.Client
}

type applicationPreviewModel struct {
	ApplicationUUID types.String `tfsdk:"application_uuid"`
	PullRequestID   types.Int64  `tfsdk:"pull_request_id"`
}

func NewResource() resource.Resource {
	return &applicationPreviewResource{}
}

func (r *applicationPreviewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_preview"
}

func (r *applicationPreviewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the lifecycle of a PR preview deployment for a Coolify application. The resource itself is state-only on create; on destroy, it deletes the preview deployment via the Coolify API.",
		Attributes: map[string]schema.Attribute{
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application that owns the preview.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"pull_request_id": schema.Int64Attribute{
				MarkdownDescription: "The pull request number for the preview deployment.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *applicationPreviewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *applicationPreviewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan applicationPreviewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{
		"resource_type": "coolify_application_preview",
		"app_uuid":      plan.ApplicationUUID.ValueString(),
		"pr_id":         plan.PullRequestID.ValueInt64(),
	})

	// Create is state-only. The preview deployment is managed by Coolify
	// (triggered by webhooks or the UI). This resource only tracks it
	// so that terraform destroy can clean it up.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *applicationPreviewResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// No read endpoint for individual previews. Preserve state.
}

func (r *applicationPreviewResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "coolify_application_preview does not support in-place updates")
}

func (r *applicationPreviewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state applicationPreviewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appUUID := state.ApplicationUUID.ValueString()
	prID := state.PullRequestID.ValueInt64()

	tflog.Debug(ctx, "deleting preview deployment", map[string]interface{}{
		"resource_type": "coolify_application_preview",
		"app_uuid":      appUUID,
		"pr_id":         prID,
	})

	if err := r.client.DeletePreviewDeployment(ctx, appUUID, prID); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError(
				"Error deleting preview deployment",
				fmt.Sprintf("Could not delete preview for application %s PR %d: %s", appUUID, prID, err),
			)
		}
	}
}

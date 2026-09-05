package applicationpreview

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

const previewDomainUpdateFloor = "Requires Coolify >= v4.3.15 (not in tag v4.3.14)."

type applicationPreviewResource struct {
	client *client.Client
}

type applicationPreviewModel struct {
	ApplicationUUID      types.String `tfsdk:"application_uuid"`
	PullRequestID        types.Int64  `tfsdk:"pull_request_id"`
	Domains              types.String `tfsdk:"domains"`
	DockerComposeDomains types.String `tfsdk:"docker_compose_domains"`
	ForceDomainOverride  types.Bool   `tfsdk:"force_domain_override"`
}

func NewResource() resource.Resource {
	return &applicationPreviewResource{}
}

func (r *applicationPreviewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_preview"
}

func (r *applicationPreviewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the lifecycle of a PR preview deployment for a Coolify application. Create is state-only unless you set preview domains; on destroy, it deletes the preview deployment via the Coolify API. There is no GET for a single preview, so domain attributes are preserved from state.",
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
			"domains": schema.StringAttribute{
				MarkdownDescription: "Comma-separated preview domain URLs for a non-compose application (for example `https://pr.example.com`). " + previewDomainUpdateFloor + " Mutually exclusive with `docker_compose_domains` on the Coolify side. Coolify has no GET for a single preview, so the value is preserved from state.",
				Optional:            true,
			},
			"docker_compose_domains": schema.StringAttribute{
				MarkdownDescription: "JSON array of `{name, domain, redirect}` objects for a Docker Compose application preview. " + previewDomainUpdateFloor + " Mutually exclusive with `domains` on the Coolify side. Coolify has no GET for a single preview, so the value is preserved from state.",
				Optional:            true,
			},
			"force_domain_override": schema.BoolAttribute{
				MarkdownDescription: "When `true`, Coolify applies the preview domains even if they conflict with another resource. Write-only; default `false`. " + previewDomainUpdateFloor,
				Optional:            true,
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

	// Create is state-only unless preview domains are set. The preview
	// deployment is created by Coolify (webhook or UI); this resource
	// tracks it so terraform destroy can clean it up, and optionally
	// PATCHes domains when Coolify >= v4.3.15.
	if plan.hasDomainWrite() {
		if err := r.patchPreviewDomains(ctx, plan); err != nil {
			resp.Diagnostics.AddError(
				"Error updating preview domains",
				fmt.Sprintf("Could not set preview domains for application %s PR %d: %s",
					plan.ApplicationUUID.ValueString(), plan.PullRequestID.ValueInt64(), err),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *applicationPreviewResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// No read endpoint for individual previews. Preserve state.
}

func (r *applicationPreviewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan applicationPreviewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.patchPreviewDomains(ctx, plan); err != nil {
		resp.Diagnostics.AddError(
			"Error updating preview domains",
			fmt.Sprintf("Could not set preview domains for application %s PR %d: %s",
				plan.ApplicationUUID.ValueString(), plan.PullRequestID.ValueInt64(), err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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

func (m applicationPreviewModel) hasDomainWrite() bool {
	if !m.Domains.IsNull() && !m.Domains.IsUnknown() {
		return true
	}
	if !m.DockerComposeDomains.IsNull() && !m.DockerComposeDomains.IsUnknown() {
		return true
	}
	return !m.ForceDomainOverride.IsNull() && !m.ForceDomainOverride.IsUnknown() && m.ForceDomainOverride.ValueBool()
}

func (r *applicationPreviewResource) patchPreviewDomains(ctx context.Context, plan applicationPreviewModel) error {
	if r.client == nil || !r.client.SupportsPreviewDomainUpdate() {
		return fmt.Errorf("preview domain updates require Coolify >= v4.3.15")
	}

	input := client.UpdatePreviewInput{}
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		v := plan.Domains.ValueString()
		input.Domains = &v
	}
	if !plan.DockerComposeDomains.IsNull() && !plan.DockerComposeDomains.IsUnknown() {
		raw := strings.TrimSpace(plan.DockerComposeDomains.ValueString())
		if raw != "" {
			if !json.Valid([]byte(raw)) {
				return fmt.Errorf("docker_compose_domains must be a JSON array")
			}
			input.DockerComposeDomains = json.RawMessage(raw)
		}
	}
	if !plan.ForceDomainOverride.IsNull() && !plan.ForceDomainOverride.IsUnknown() && plan.ForceDomainOverride.ValueBool() {
		v := true
		input.ForceDomainOverride = &v
	}

	return r.client.UpdatePreviewDeployment(ctx, plan.ApplicationUUID.ValueString(), plan.PullRequestID.ValueInt64(), input)
}

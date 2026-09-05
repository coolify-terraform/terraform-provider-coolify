package applicationpreview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
		MarkdownDescription: "Tracks a Coolify PR preview so terraform destroy can delete it, and optionally PATCHes domains when that preview already exists. This resource does **not** create the PR preview; Coolify creates it (webhook or UI). There is no GET for a single preview, so domain attributes are preserved from state. Apply returns 404 if Coolify has no preview for the pull request yet.",
		Attributes: map[string]schema.Attribute{
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application that owns the preview.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"pull_request_id": schema.Int64Attribute{
				MarkdownDescription: "The pull request number for the preview deployment. Must be a positive integer (Coolify 422s 0).",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"domains": schema.StringAttribute{
				MarkdownDescription: "Comma-separated preview domain URLs for a non-compose application (for example `https://pr.example.com`). PATCHes an **existing** preview; Coolify returns 404 if it has not created the preview yet. " + previewDomainUpdateFloor + " Mutually exclusive with `docker_compose_domains` on the Coolify side. Coolify has no GET for a single preview, so the value is preserved from state. An empty string skips PATCH; it does not clear Coolify preview domains.",
				Optional:            true,
				Validators: []validator.String{
					validate.Domains(),
					stringvalidator.ConflictsWith(path.MatchRoot("docker_compose_domains")),
				},
			},
			"docker_compose_domains": schema.StringAttribute{
				MarkdownDescription: "JSON array of `{name, domain, redirect}` objects for a Docker Compose application preview. Extra keys are rejected at plan; `redirect` must be `www`, `non-www`, or `both`. PATCHes an **existing** preview; Coolify returns 404 if it has not created the preview yet. " + previewDomainUpdateFloor + " Mutually exclusive with `domains` on the Coolify side. Coolify has no GET for a single preview, so the value is preserved from state. An empty string and `[]` skip PATCH; they do not clear Coolify preview domains.",
				Optional:            true,
				Validators: []validator.String{
					validate.DockerComposeDomains(),
					stringvalidator.ConflictsWith(path.MatchRoot("domains")),
				},
			},
			"force_domain_override": schema.BoolAttribute{
				MarkdownDescription: "When `true`, Coolify applies the preview domains even if they conflict with another resource. Omitted or `false` is not sent (Coolify keeps its default). Coolify has no GET for a single preview, so the value is preserved from state. " + previewDomainUpdateFloor,
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
	input, ok, err := plan.previewDomainInput()
	if err != nil {
		resp.Diagnostics.AddError(previewDomainError(plan, err))
		return
	}
	if ok {
		if err := r.patchPreviewDomains(ctx, plan, input); err != nil {
			resp.Diagnostics.AddError(previewDomainError(plan, err))
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

	input, ok, err := plan.previewDomainInput()
	if err != nil {
		resp.Diagnostics.AddError(previewDomainError(plan, err))
		return
	}
	if ok {
		if err := r.patchPreviewDomains(ctx, plan, input); err != nil {
			resp.Diagnostics.AddError(previewDomainError(plan, err))
			return
		}
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

func hasNonEmptyDomainSegment(raw string) bool {
	for _, part := range strings.Split(strings.TrimSpace(raw), ",") {
		if strings.TrimSpace(part) != "" {
			return true
		}
	}
	return false
}

func (m applicationPreviewModel) previewDomainInput() (client.UpdatePreviewInput, bool, error) {
	input := client.UpdatePreviewInput{}
	ok := false

	if !m.Domains.IsNull() && !m.Domains.IsUnknown() && hasNonEmptyDomainSegment(m.Domains.ValueString()) {
		v := strings.TrimSpace(m.Domains.ValueString())
		input.Domains = &v
		ok = true
	}

	if !m.DockerComposeDomains.IsNull() && !m.DockerComposeDomains.IsUnknown() {
		raw := strings.TrimSpace(m.DockerComposeDomains.ValueString())
		if raw != "" {
			var items []json.RawMessage
			if err := json.Unmarshal([]byte(raw), &items); err != nil {
				return client.UpdatePreviewInput{}, false, fmt.Errorf("docker_compose_domains must be a JSON array: %w", err)
			}
			if len(items) > 0 {
				input.DockerComposeDomains = json.RawMessage(raw)
				ok = true
			}
		}
	}

	if ok && !m.ForceDomainOverride.IsNull() && !m.ForceDomainOverride.IsUnknown() && m.ForceDomainOverride.ValueBool() {
		v := true
		input.ForceDomainOverride = &v
	}

	return input, ok, nil
}

func (r *applicationPreviewResource) patchPreviewDomains(ctx context.Context, plan applicationPreviewModel, input client.UpdatePreviewInput) error {
	if r.client == nil || !r.client.SupportsPreviewDomainUpdate() {
		return fmt.Errorf("preview domain updates require Coolify >= v4.3.15")
	}
	return r.client.UpdatePreviewDeployment(ctx, plan.ApplicationUUID.ValueString(), plan.PullRequestID.ValueInt64(), input)
}

func previewDomainError(plan applicationPreviewModel, err error) (string, string) {
	return "Error updating preview domains",
		fmt.Sprintf("Could not set preview domains for application %s PR %d: %s",
			plan.ApplicationUUID.ValueString(), plan.PullRequestID.ValueInt64(),
			annotatePreviewDomainError(err))
}

func annotatePreviewDomainError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case previewLooksMissing(err):
		return msg + ". Coolify has no preview for this PR yet. Open the PR or trigger a preview deploy, then re-apply."
	case previewLooksConflict(err):
		return msg + ". set force_domain_override = true if you intend to take over the domain."
	default:
		return msg
	}
}

func previewLooksMissing(err error) bool {
	if client.IsNotFound(err) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "preview not found") || strings.Contains(lower, "status 404")
}

func previewLooksConflict(err error) bool {
	var apiErr *client.APIStatusError
	if errors.As(err, &apiErr) && apiErr.Status == 409 {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "status 409") || strings.Contains(lower, "conflict")
}

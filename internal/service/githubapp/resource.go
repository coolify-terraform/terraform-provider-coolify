package githubapp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*gitHubAppResource)(nil)
	_ resource.ResourceWithImportState = (*gitHubAppResource)(nil)
	_ resource.ResourceWithConfigure   = (*gitHubAppResource)(nil)
)

// gitHubAppResource is the resource implementation for a Coolify GitHub App integration.
type gitHubAppResource struct {
	client *client.Client
}

// gitHubAppResourceModel maps the resource schema data.
type gitHubAppResourceModel struct {
	ID               types.Int64  `tfsdk:"id"`
	UUID             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	OrganizationName types.String `tfsdk:"organization_name"`
	AppID            types.Int64  `tfsdk:"app_id"`
	InstallationID   types.Int64  `tfsdk:"installation_id"`
	ClientID         types.String `tfsdk:"client_id"`
	ClientSecret     types.String `tfsdk:"client_secret"`
	WebhookSecret    types.String `tfsdk:"webhook_secret"`
	PrivateKeyUUID   types.String `tfsdk:"private_key_uuid"`
}

// NewResource returns a new GitHub App resource instance.
func NewResource() resource.Resource {
	return &gitHubAppResource{}
}

func (r *gitHubAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_app"
}

func (r *gitHubAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify GitHub App integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The numeric identifier of the GitHub App.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the GitHub App.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the GitHub App.",
				Required:            true,
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "The GitHub organization name.",
				Optional:            true,
				Computed:            true,
			},
			"app_id": schema.Int64Attribute{
				MarkdownDescription: "The GitHub App ID.",
				Required:            true,
			},
			"installation_id": schema.Int64Attribute{
				MarkdownDescription: "The GitHub App installation ID.",
				Required:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The GitHub App client ID.",
				Required:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The GitHub App client secret. Write-only: not returned by the API after creation.",
				Required:            true,
				Sensitive:           true,
			},
			"webhook_secret": schema.StringAttribute{
				MarkdownDescription: "The GitHub App webhook secret. If omitted on create, the provider sends `<name>-webhook` after trimming surrounding whitespace from `name`, or `terraform-provider-coolify` when `name` is empty.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
			"private_key_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of an existing `coolify_private_key` resource for GitHub App authentication. Write-only: not returned by the API after creation.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *gitHubAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *gitHubAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gitHubAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_github_app"})

	input := client.CreateGitHubAppIntegrationInput{
		Name:           plan.Name.ValueString(),
		APIURL:         "https://api.github.com",
		HTMLURL:        "https://github.com",
		AppID:          plan.AppID.ValueInt64(),
		InstallationID: plan.InstallationID.ValueInt64(),
		ClientID:       plan.ClientID.ValueString(),
		ClientSecret:   plan.ClientSecret.ValueString(),
		PrivateKeyUUID: plan.PrivateKeyUUID.ValueString(),
	}
	flex.SetIfKnown(&input.OrganizationName, plan.OrganizationName)
	if plan.WebhookSecret.IsNull() || plan.WebhookSecret.IsUnknown() {
		input.WebhookSecret = defaultWebhookSecret(plan.Name.ValueString())
	} else {
		flex.SetIfKnown(&input.WebhookSecret, plan.WebhookSecret)
	}

	app, err := r.client.CreateGitHubApp(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitHub App", err.Error())
		return
	}

	plan.ID = types.Int64Value(app.ID)
	plan.UUID = flex.StringToFramework(app.UUID)
	if plan.OrganizationName.IsUnknown() {
		plan.OrganizationName = types.StringNull()
	}
	if plan.WebhookSecret.IsUnknown() {
		plan.WebhookSecret = types.StringNull()
	}

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back the full object to populate all fields.
	diags := r.readGitHubApp(ctx, app.ID, &plan)
	if diags.HasError() {
		resp.Diagnostics.AddError(
			"GitHub App created but refresh failed",
			fmt.Sprintf("Coolify created GitHub App %d, but the provider could not read it back: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", app.ID, diags.Errors()[0].Detail()),
		)
		return
	}
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitHubAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gitHubAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_github_app", "id": state.ID.ValueInt64()})

	app, err := r.client.GetGitHubApp(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			// The GitHub App was deleted outside of Terraform; remove from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading GitHub App", fmt.Sprintf("Could not read GitHub App %d: %s", state.ID.ValueInt64(), err))
		return
	}

	flattenGitHubApp(app, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *gitHubAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gitHubAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state gitHubAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_github_app", "id": state.ID.ValueInt64()})

	input := client.UpdateGitHubAppIntegrationInput{
		Name:             flex.StringIfChanged(plan.Name, state.Name),
		OrganizationName: flex.StringIfChanged(plan.OrganizationName, state.OrganizationName),
		AppID:            flex.Int64IfChanged(plan.AppID, state.AppID),
		InstallationID:   flex.Int64IfChanged(plan.InstallationID, state.InstallationID),
		ClientID:         flex.StringIfChanged(plan.ClientID, state.ClientID),
		ClientSecret:     flex.StringIfChanged(plan.ClientSecret, state.ClientSecret),
		WebhookSecret:    flex.StringIfChanged(plan.WebhookSecret, state.WebhookSecret),
		PrivateKeyUUID:   flex.StringIfChanged(plan.PrivateKeyUUID, state.PrivateKeyUUID),
	}

	// Use the PATCH response directly (returns the full object) instead of
	// a separate GET read-back. This avoids the O(n) list-and-scan in
	// GetGitHubApp since the Coolify API has no GET /github-apps/{id}.
	app, err := r.client.UpdateGitHubApp(ctx, state.ID.ValueInt64(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating GitHub App", fmt.Sprintf("Could not update GitHub App %d: %s", state.ID.ValueInt64(), err))
		return
	}

	flattenGitHubApp(app, &plan)
	if plan.WebhookSecret.IsNull() || plan.WebhookSecret.IsUnknown() {
		plan.WebhookSecret = state.WebhookSecret
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitHubAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gitHubAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_github_app", "id": state.ID.ValueInt64()})

	err := r.client.DeleteGitHubApp(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			// Already deleted; nothing to do.
			return
		}
		resp.Diagnostics.AddError("Error deleting GitHub App", err.Error())
	}
}

func (r *gitHubAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Could not parse GitHub App ID %q as integer: %s", req.ID, err),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// readGitHubApp fetches the GitHub App from the API and updates the model in place.
// client_secret and private_key_uuid are write-only and preserved from the caller's model.
func (r *gitHubAppResource) readGitHubApp(ctx context.Context, id int64, model *gitHubAppResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	app, err := r.client.GetGitHubApp(ctx, id)
	if err != nil {
		diags.AddError("Error reading GitHub App", fmt.Sprintf("Could not read GitHub App %d: %s", id, err))
		return diags
	}

	flattenGitHubApp(app, model)
	return diags
}

// flattenGitHubApp maps API fields into the Terraform resource model.
// client_secret and private_key_uuid are write-only (not returned by API)
// and preserved from the plan/state by the caller.
func flattenGitHubApp(app *client.GitHubApp, model *gitHubAppResourceModel) {
	model.ID = types.Int64Value(app.ID)
	model.UUID = flex.StringToFramework(app.UUID)
	model.Name = types.StringValue(app.Name)
	model.OrganizationName = flex.StringToFramework(app.OrganizationName)
	model.AppID = types.Int64Value(app.AppID)
	model.InstallationID = types.Int64Value(app.InstallationID)
	model.ClientID = types.StringValue(app.ClientID)
	if app.WebhookSecret != "" {
		model.WebhookSecret = types.StringValue(app.WebhookSecret)
	}
}

func defaultWebhookSecret(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "terraform-provider-coolify"
	}
	return trimmed + "-webhook"
}

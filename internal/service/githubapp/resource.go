package githubapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
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
	_ resource.Resource                 = (*gitHubAppResource)(nil)
	_ resource.ResourceWithImportState  = (*gitHubAppResource)(nil)
	_ resource.ResourceWithConfigure    = (*gitHubAppResource)(nil)
	_ resource.ResourceWithUpgradeState = (*gitHubAppResource)(nil)
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
		Version:             1,
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
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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
				MarkdownDescription: "The GitHub App webhook secret. If omitted on create, the provider generates a random secret, sends it to Coolify, and stores it in state. Coolify does not reliably return it after create or import, so keep the value in your Terraform configuration before the first terraform plan after import.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
			"private_key_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of an existing `coolify_private_key` resource for GitHub App authentication. Write-only: not returned by the API after creation.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
		},
	}
}

func (r *gitHubAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
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
		generatedSecret, err := randomWebhookSecret()
		if err != nil {
			resp.Diagnostics.AddError("Error generating GitHub App webhook secret", err.Error())
			return
		}
		input.WebhookSecret = generatedSecret
	} else {
		flex.SetIfKnown(&input.WebhookSecret, plan.WebhookSecret)
	}

	app, err := r.client.CreateGitHubApp(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitHub App", fmt.Sprintf("github app %q: %s", plan.Name.ValueString(), err))
		return
	}

	// Coolify has no GET /github-apps/{id} route, and POST already returns the
	// created object, so avoid the extra list-backed refresh here.
	flattenGitHubApp(app, &plan)

	// Coolify may omit webhook_secret in responses. Preserve the value we sent
	// so create still converges in one apply.
	if plan.WebhookSecret.IsNull() || plan.WebhookSecret.IsUnknown() {
		plan.WebhookSecret = types.StringValue(input.WebhookSecret)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_github_app", "uuid": plan.UUID.ValueString()})
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
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_github_app", "uuid": state.UUID.ValueString()})
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
		resp.Diagnostics.AddError("Error deleting GitHub App", fmt.Sprintf("Could not delete GitHub App %d: %s", state.ID.ValueInt64(), err))
		return
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": "coolify_github_app", "uuid": state.UUID.ValueString()})
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

func (r *gitHubAppResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			// Version 0 -> 1: rename private_key (raw PEM content) to
			// private_key_uuid (UUID reference to coolify_private_key).
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":                schema.Int64Attribute{Computed: true},
					"uuid":              schema.StringAttribute{Computed: true},
					"name":              schema.StringAttribute{Required: true},
					"organization_name": schema.StringAttribute{Optional: true, Computed: true},
					"app_id":            schema.Int64Attribute{Required: true},
					"installation_id":   schema.Int64Attribute{Required: true},
					"client_id":         schema.StringAttribute{Required: true},
					"client_secret":     schema.StringAttribute{Required: true, Sensitive: true},
					"webhook_secret":    schema.StringAttribute{Optional: true, Computed: true, Sensitive: true},
					"private_key":       schema.StringAttribute{Required: true, Sensitive: true},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				type v0Model struct {
					ID               types.Int64  `tfsdk:"id"`
					UUID             types.String `tfsdk:"uuid"`
					Name             types.String `tfsdk:"name"`
					OrganizationName types.String `tfsdk:"organization_name"`
					AppID            types.Int64  `tfsdk:"app_id"`
					InstallationID   types.Int64  `tfsdk:"installation_id"`
					ClientID         types.String `tfsdk:"client_id"`
					ClientSecret     types.String `tfsdk:"client_secret"`
					WebhookSecret    types.String `tfsdk:"webhook_secret"`
					PrivateKey       types.String `tfsdk:"private_key"`
				}
				var old v0Model
				resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
				if resp.Diagnostics.HasError() {
					return
				}
				resp.Diagnostics.AddWarning(
					"State migrated: private_key renamed to private_key_uuid",
					"The coolify_github_app resource renamed private_key (raw PEM content) to private_key_uuid "+
						"(UUID of a coolify_private_key resource). The old value cannot be automatically "+
						"converted. Update your configuration to set private_key_uuid to the UUID of an "+
						"existing coolify_private_key resource, then run terraform apply.",
				)
				resp.Diagnostics.Append(resp.State.Set(ctx, &gitHubAppResourceModel{
					ID: old.ID, UUID: old.UUID, Name: old.Name,
					OrganizationName: old.OrganizationName,
					AppID:            old.AppID, InstallationID: old.InstallationID,
					ClientID: old.ClientID, ClientSecret: old.ClientSecret,
					WebhookSecret: old.WebhookSecret,
					// Cannot convert raw PEM content to UUID; user must update config.
					PrivateKeyUUID: types.StringUnknown(),
				})...)
			},
		},
	}
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

func randomWebhookSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating random webhook secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

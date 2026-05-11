package githubapp

import (
	"context"
	"fmt"
	"strconv"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	Name             types.String `tfsdk:"name"`
	OrganizationName types.String `tfsdk:"organization_name"`
	AppID            types.Int64  `tfsdk:"app_id"`
	InstallationID   types.Int64  `tfsdk:"installation_id"`
	ClientID         types.String `tfsdk:"client_id"`
	ClientSecret     types.String `tfsdk:"client_secret"`
	WebhookSecret    types.String `tfsdk:"webhook_secret"`
	PrivateKey       types.String `tfsdk:"private_key"`
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
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the GitHub App.",
				Required:            true,
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "The GitHub organization name.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"app_id": schema.Int64Attribute{
				MarkdownDescription: "The GitHub App ID.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"installation_id": schema.Int64Attribute{
				MarkdownDescription: "The GitHub App installation ID.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The GitHub App client ID.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The GitHub App client secret.",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"webhook_secret": schema.StringAttribute{
				MarkdownDescription: "The GitHub App webhook secret.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "The GitHub App private key.",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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

	input := client.CreateGitHubAppIntegrationInput{
		Name:           plan.Name.ValueString(),
		AppID:          plan.AppID.ValueInt64(),
		InstallationID: plan.InstallationID.ValueInt64(),
		ClientID:       plan.ClientID.ValueString(),
		ClientSecret:   plan.ClientSecret.ValueString(),
		PrivateKey:     plan.PrivateKey.ValueString(),
	}
	flex.SetIfKnown(&input.OrganizationName, plan.OrganizationName)
	flex.SetIfKnown(&input.WebhookSecret, plan.WebhookSecret)

	app, err := r.client.CreateGitHubApp(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating GitHub App", fmt.Sprintf("Could not create GitHub App: %s", err))
		return
	}

	plan.ID = types.Int64Value(app.ID)

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back the full object to populate all fields.
	diags := r.readGitHubApp(ctx, app.ID, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitHubAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gitHubAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetGitHubApp(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			// The GitHub App was deleted outside of Terraform; remove from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading GitHub App", fmt.Sprintf("Could not read GitHub App: %s", err))
		return
	}

	state.ID = types.Int64Value(app.ID)
	state.Name = types.StringValue(app.Name)
	state.OrganizationName = flex.StringToFramework(app.OrganizationName)
	state.AppID = types.Int64Value(app.AppID)
	state.InstallationID = types.Int64Value(app.InstallationID)
	state.ClientID = types.StringValue(app.ClientID)
	state.WebhookSecret = flex.StringToFramework(app.WebhookSecret)
	// client_secret and private_key are write-only; preserved from state.

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

	input := client.UpdateGitHubAppIntegrationInput{}
	flex.SetStrPtr(&input.Name, plan.Name)
	flex.SetStrPtr(&input.WebhookSecret, plan.WebhookSecret)

	_, err := r.client.UpdateGitHubApp(ctx, state.ID.ValueInt64(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating GitHub App", fmt.Sprintf("Could not update GitHub App: %s", err))
		return
	}

	plan.ID = state.ID

	// Read back the full object to populate all fields.
	diags := r.readGitHubApp(ctx, state.ID.ValueInt64(), &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitHubAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gitHubAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteGitHubApp(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			// Already deleted; nothing to do.
			return
		}
		resp.Diagnostics.AddError("Error Deleting GitHub App", fmt.Sprintf("Could not delete GitHub App: %s", err))
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
// client_secret and private_key are write-only and preserved from the caller's model.
func (r *gitHubAppResource) readGitHubApp(ctx context.Context, id int64, model *gitHubAppResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	app, err := r.client.GetGitHubApp(ctx, id)
	if err != nil {
		diags.AddError("Error Reading GitHub App", fmt.Sprintf("Could not read GitHub App after create/update: %s", err))
		return diags
	}

	model.ID = types.Int64Value(app.ID)
	model.Name = types.StringValue(app.Name)
	model.OrganizationName = flex.StringToFramework(app.OrganizationName)
	model.AppID = types.Int64Value(app.AppID)
	model.InstallationID = types.Int64Value(app.InstallationID)
	model.ClientID = types.StringValue(app.ClientID)
	model.WebhookSecret = flex.StringToFramework(app.WebhookSecret)
	// client_secret and private_key are write-only; preserved from state/plan.

	return diags
}

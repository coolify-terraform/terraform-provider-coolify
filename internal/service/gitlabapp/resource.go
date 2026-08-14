package gitlabapp

import (
	"context"
	"fmt"
	"strconv"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*gitlabAppResource)(nil)
	_ resource.ResourceWithConfigure   = (*gitlabAppResource)(nil)
	_ resource.ResourceWithImportState = (*gitlabAppResource)(nil)
)

type gitlabAppResource struct{ client *client.Client }

type gitlabAppModel struct {
	ID           types.Int64  `tfsdk:"id"`
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	HTMLURL      types.String `tfsdk:"html_url"`
	APIURL       types.String `tfsdk:"api_url"`
	CustomUser   types.String `tfsdk:"custom_user"`
	CustomPort   types.Int64  `tfsdk:"custom_port"`
	GroupName    types.String `tfsdk:"group_name"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	WebhookToken types.String `tfsdk:"webhook_token"`
	RedirectURI  types.String `tfsdk:"redirect_uri"`
	IsSystemWide types.Bool   `tfsdk:"is_system_wide"`
}

func NewResource() resource.Resource { return &gitlabAppResource{} }

func (r *gitlabAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gitlab_app"
}

func (r *gitlabAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coolify GitLab App source. Requires Coolify >= v4.3.0.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.Int64Attribute{Computed: true, MarkdownDescription: "Coolify numeric GitLab App id (used in PATCH/DELETE)."},
			"uuid":           schema.StringAttribute{Computed: true, MarkdownDescription: "Coolify UUID when returned.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":           schema.StringAttribute{Required: true, MarkdownDescription: "Display name."},
			"html_url":       schema.StringAttribute{Required: true, MarkdownDescription: "GitLab HTML URL (for example https://gitlab.example.com)."},
			"api_url":        schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "GitLab API URL. Coolify derives it from html_url when omitted.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"custom_user":    schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "SSH user. Coolify defaults to git.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"custom_port":    schema.Int64Attribute{Optional: true, Computed: true, MarkdownDescription: "SSH port. Coolify defaults to 22.", PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"group_name":     schema.StringAttribute{Optional: true, MarkdownDescription: "Optional group filter."},
			"client_id":      schema.StringAttribute{Optional: true, MarkdownDescription: "OAuth application id."},
			"client_secret":  schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "OAuth application secret. Preserved when GET omits it."},
			"webhook_token":  schema.StringAttribute{Optional: true, Computed: true, Sensitive: true, MarkdownDescription: "Webhook secret. Coolify generates one when omitted.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"redirect_uri":   schema.StringAttribute{Optional: true, MarkdownDescription: "OAuth redirect URI. May be a private URL."},
			"is_system_wide": schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: "System-wide on self-hosted Coolify only.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *gitlabAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func flattenGitLab(a *client.GitLabApp, m *gitlabAppModel) {
	m.ID = types.Int64Value(a.ID)
	if a.UUID != "" {
		m.UUID = types.StringValue(a.UUID)
	}
	m.Name = types.StringValue(a.Name)
	m.HTMLURL = types.StringValue(a.HTMLURL)
	if a.APIURL != "" {
		m.APIURL = types.StringValue(a.APIURL)
	}
	if a.CustomUser != "" {
		m.CustomUser = types.StringValue(a.CustomUser)
	} else if m.CustomUser.IsNull() || m.CustomUser.IsUnknown() {
		m.CustomUser = types.StringValue("git")
	}
	if a.CustomPort != nil {
		m.CustomPort = types.Int64Value(*a.CustomPort)
	} else if m.CustomPort.IsNull() || m.CustomPort.IsUnknown() {
		m.CustomPort = types.Int64Value(22)
	}
	if a.GroupName != "" {
		m.GroupName = types.StringValue(a.GroupName)
	}
	if a.ClientID != "" {
		m.ClientID = types.StringValue(a.ClientID)
	}
	if a.ClientSecret != "" {
		m.ClientSecret = types.StringValue(a.ClientSecret)
	}
	if a.WebhookToken != "" {
		m.WebhookToken = types.StringValue(a.WebhookToken)
	} else if m.WebhookToken.IsUnknown() {
		m.WebhookToken = types.StringValue("")
	}
	if a.RedirectURI != "" {
		m.RedirectURI = types.StringValue(a.RedirectURI)
	}
	m.IsSystemWide = types.BoolValue(a.IsSystemWide)
}

func (r *gitlabAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gitlabAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input := client.CreateGitLabAppInput{
		Name: plan.Name.ValueString(), HTMLURL: plan.HTMLURL.ValueString(),
		APIURL: plan.APIURL.ValueString(), CustomUser: plan.CustomUser.ValueString(),
		GroupName: plan.GroupName.ValueString(), ClientID: plan.ClientID.ValueString(),
		ClientSecret: plan.ClientSecret.ValueString(), WebhookToken: plan.WebhookToken.ValueString(),
		RedirectURI: plan.RedirectURI.ValueString(),
	}
	if !plan.CustomPort.IsNull() && !plan.CustomPort.IsUnknown() {
		v := plan.CustomPort.ValueInt64()
		input.CustomPort = &v
	}
	if !plan.IsSystemWide.IsNull() && !plan.IsSystemWide.IsUnknown() {
		v := plan.IsSystemWide.ValueBool()
		input.IsSystemWide = &v
	}
	created, err := r.client.CreateGitLabApp(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating GitLab App", err.Error())
		return
	}
	flattenGitLab(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitlabAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gitlabAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.GetGitLabApp(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading GitLab App", err.Error())
		return
	}
	secret, token := state.ClientSecret, state.WebhookToken
	flattenGitLab(got, &state)
	if got.ClientSecret == "" {
		state.ClientSecret = secret
	}
	if got.WebhookToken == "" {
		state.WebhookToken = token
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func knownString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func updateInputFromPlan(plan gitlabAppModel) client.UpdateGitLabAppInput {
	name := plan.Name.ValueString()
	html := plan.HTMLURL.ValueString()
	input := client.UpdateGitLabAppInput{Name: &name, HTMLURL: &html}
	input.APIURL = knownString(plan.APIURL)
	input.CustomUser = knownString(plan.CustomUser)
	input.GroupName = knownString(plan.GroupName)
	input.ClientID = knownString(plan.ClientID)
	input.ClientSecret = knownString(plan.ClientSecret)
	input.WebhookToken = knownString(plan.WebhookToken)
	input.RedirectURI = knownString(plan.RedirectURI)
	if !plan.CustomPort.IsNull() && !plan.CustomPort.IsUnknown() {
		v := plan.CustomPort.ValueInt64()
		input.CustomPort = &v
	}
	return input
}

func (r *gitlabAppResource) resolveUpdateID(ctx context.Context, plan *gitlabAppModel) int64 {
	id := plan.ID.ValueInt64()
	if id != 0 {
		return id
	}
	got, err := r.client.GetGitLabAppByUUID(ctx, plan.UUID.ValueString())
	if err != nil || got.ID == 0 {
		return id
	}
	plan.ID = types.Int64Value(got.ID)
	return got.ID
}

func preserveGitLabSecrets(got *client.GitLabApp, plan *gitlabAppModel, secret, token types.String) {
	if got.ClientSecret == "" {
		plan.ClientSecret = secret
	}
	if got.WebhookToken == "" {
		plan.WebhookToken = token
	}
}

func (r *gitlabAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gitlabAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	secret, token := plan.ClientSecret, plan.WebhookToken
	got, err := r.client.UpdateGitLabApp(ctx, r.resolveUpdateID(ctx, &plan), updateInputFromPlan(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating GitLab App", err.Error())
		return
	}
	flattenGitLab(got, &plan)
	preserveGitLabSecrets(got, &plan, secret, token)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gitlabAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gitlabAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteGitLabApp(ctx, state.ID.ValueInt64()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting GitLab App", err.Error())
	}
}

func (r *gitlabAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if id, err := strconv.ParseInt(req.ID, 10, 64); err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}
	got, err := r.client.GetGitLabAppByUUID(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("expected numeric id or uuid: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), got.ID)...)
}

package notificationslack

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*slackResource)(nil)
	_ resource.ResourceWithConfigure   = (*slackResource)(nil)
	_ resource.ResourceWithImportState = (*slackResource)(nil)
)

type slackResource struct {
	client *client.Client
}

type model struct {
	ID         types.String `tfsdk:"id"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	WebhookURL types.String `tfsdk:"webhook_url"`

	notificationcommon.EventModel
}

// NewResource returns a new Slack notification resource.
func NewResource() resource.Resource {
	return &slackResource{}
}

func (r *slackResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_slack"
}

func (r *slackResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id":      notificationcommon.IDAttribute(),
		"enabled": notificationcommon.EnabledAttribute("Slack"),
		"webhook_url": schema.StringAttribute{
			MarkdownDescription: "Slack incoming webhook URL. Sensitive; Coolify may omit it on read unless the API token " +
				"can read sensitive fields (`read:sensitive` or root). Preserve the value in configuration after import. " +
				"Must pass Coolify's SafeWebhookUrl rule (public http/https; private/loopback hosts rejected).",
			Optional:  true,
			Computed:  true,
			Sensitive: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the current team's Slack notification settings in Coolify. " +
			"This is a team-scoped singleton (one configuration per team, selected by the API token). " +
			"Requires Coolify >= v4.3.0.\n\n" +
			"On destroy, Slack notifications are disabled (`enabled = false`); the webhook URL is left unchanged. " +
			"Import with id `current`.",
		Attributes: notificationcommon.MergeAttrs(attrs, notificationcommon.EventSchemaAttrs("Slack")),
	}
}

func (r *slackResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *slackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_notification_slack"})

	updated, err := r.client.UpdateSlackNotifications(ctx, createInputFromPlan(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error configuring Slack notifications", err.Error())
		return
	}

	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *slackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_notification_slack"})

	got, err := r.client.GetSlackNotifications(ctx)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Slack notifications", err.Error())
		return
	}

	flatten(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *slackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_notification_slack"})

	updated, err := r.client.UpdateSlackNotifications(ctx, updateInputFromPlan(plan, state))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Slack notifications", err.Error())
		return
	}

	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *slackResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "deleting resource (disabling Slack notifications)", map[string]interface{}{"resource_type": "coolify_notification_slack"})
	enabled := false
	_, err := r.client.UpdateSlackNotifications(ctx, client.UpdateSlackNotificationInput{
		Enabled: &enabled,
	})
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error disabling Slack notifications on destroy", err.Error())
	}
}

func (r *slackResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	notificationcommon.ImportStateCurrent(ctx, req, resp, "coolify_notification_slack")
}

func createInputFromPlan(plan model) client.UpdateSlackNotificationInput {
	in := client.UpdateSlackNotificationInput{
		Enabled: flex.BoolValueOrNull(plan.Enabled),
	}
	_ = client.ApplyEventUpdate(&in, plan.CreateUpdate())
	if flex.StringValueConfigured(plan.WebhookURL) {
		v := plan.WebhookURL.ValueString()
		in.Webhook = &v
	}
	return in
}

func updateInputFromPlan(plan, state model) client.UpdateSlackNotificationInput {
	in := client.UpdateSlackNotificationInput{
		Enabled: flex.BoolIfChanged(plan.Enabled, state.Enabled),
	}
	_ = client.ApplyEventUpdate(&in, plan.DiffUpdate(state.EventModel))
	if w := flex.StringIfChanged(plan.WebhookURL, state.WebhookURL); w != nil {
		in.Webhook = w
	}
	return in
}

func flatten(api *client.SlackNotificationSettings, m *model) {
	if ev, err := client.EventsFrom(api); err == nil {
		m.FlattenEvents(ev)
	}
	m.ID = types.StringValue(notificationcommon.ImportIDCurrent)
	m.Enabled = types.BoolValue(api.Enabled)
	flex.SetStringPreserveEmpty(&m.WebhookURL, api.Webhook)
}

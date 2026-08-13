package notificationwebhook

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
	_ resource.Resource                = (*webhookResource)(nil)
	_ resource.ResourceWithConfigure   = (*webhookResource)(nil)
	_ resource.ResourceWithImportState = (*webhookResource)(nil)
)

type webhookResource struct {
	client *client.Client
}

type model struct {
	ID         types.String `tfsdk:"id"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	WebhookURL types.String `tfsdk:"webhook_url"`

	notificationcommon.EventModel
}

// NewResource returns a new Webhook notification resource.
func NewResource() resource.Resource {
	return &webhookResource{}
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id":      notificationcommon.IDAttribute(),
		"enabled": notificationcommon.EnabledAttribute("Webhook"),
		"webhook_url": schema.StringAttribute{
			MarkdownDescription: "Generic webhook URL. Sensitive; Coolify may omit it on read unless the API token " +
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
		MarkdownDescription: "Manages the current team's Webhook notification settings in Coolify. " +
			"This is a team-scoped singleton (one configuration per team, selected by the API token). " +
			"Requires Coolify >= v4.3.0.\n\n" +
			"On destroy, Webhook notifications are disabled (`enabled = false`); the webhook URL is left unchanged. " +
			"Import with id `current`.",
		Attributes: notificationcommon.MergeAttrs(attrs, notificationcommon.EventSchemaAttrs("Webhook")),
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_notification_webhook"})

	input, err := createInputFromPlan(plan)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	updated, err := r.client.UpdateWebhookNotifications(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error configuring Webhook notifications", err.Error())
		return
	}

	if err := flatten(updated, &plan); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_notification_webhook"})

	got, err := r.client.GetWebhookNotifications(ctx)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Webhook notifications", err.Error())
		return
	}

	if err := flatten(got, &state); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_notification_webhook"})

	input, err := updateInputFromPlan(plan, state)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	updated, err := r.client.UpdateWebhookNotifications(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Webhook notifications", err.Error())
		return
	}

	if err := flatten(updated, &plan); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "deleting resource (disabling Webhook notifications)", map[string]interface{}{"resource_type": "coolify_notification_webhook"})
	enabled := false
	_, err := r.client.UpdateWebhookNotifications(ctx, client.UpdateWebhookNotificationInput{
		Enabled: &enabled,
	})
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error disabling Webhook notifications on destroy", err.Error())
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	notificationcommon.ImportStateCurrent(ctx, req, resp, "coolify_notification_webhook")
}

func createInputFromPlan(plan model) (client.UpdateWebhookNotificationInput, error) {
	in := client.UpdateWebhookNotificationInput{
		Enabled: flex.BoolValueOrNull(plan.Enabled),
	}
	if err := client.ApplyEventUpdate(&in, plan.CreateUpdate()); err != nil {
		return in, err
	}
	if flex.StringValueConfigured(plan.WebhookURL) {
		v := plan.WebhookURL.ValueString()
		in.Webhook = &v
	}
	return in, nil
}

func updateInputFromPlan(plan, state model) (client.UpdateWebhookNotificationInput, error) {
	in := client.UpdateWebhookNotificationInput{
		Enabled: flex.BoolIfChanged(plan.Enabled, state.Enabled),
	}
	if err := client.ApplyEventUpdate(&in, plan.DiffUpdate(state.EventModel)); err != nil {
		return in, err
	}
	if w := flex.StringIfChanged(plan.WebhookURL, state.WebhookURL); w != nil {
		in.Webhook = w
	}
	return in, nil
}

func flatten(api *client.WebhookNotificationSettings, m *model) error {
	ev, err := client.EventsFrom(api)
	if err != nil {
		return err
	}
	m.FlattenEvents(ev)
	m.ID = types.StringValue(notificationcommon.ImportIDCurrent)
	m.Enabled = types.BoolValue(api.Enabled)
	flex.SetStringPreserveEmpty(&m.WebhookURL, api.Webhook)
	return nil
}

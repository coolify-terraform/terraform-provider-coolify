package notificationtelegram

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
	_ resource.Resource                = (*telegramResource)(nil)
	_ resource.ResourceWithConfigure   = (*telegramResource)(nil)
	_ resource.ResourceWithImportState = (*telegramResource)(nil)
)

type telegramResource struct {
	client *client.Client
}

type model struct {
	ID      types.String `tfsdk:"id"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Token   types.String `tfsdk:"token"`
	ChatID  types.String `tfsdk:"chat_id"`

	notificationcommon.EventModel
	notificationcommon.ThreadModel
}

func NewResource() resource.Resource {
	return &telegramResource{}
}

func (r *telegramResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_telegram"
}

func (r *telegramResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	sensStr := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: desc + " Sensitive; Coolify may omit it on read unless the API token can read sensitive fields (`read:sensitive` or root). Preserve after import.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		}
	}
	attrs := map[string]schema.Attribute{
		"id":      notificationcommon.IDAttribute(),
		"enabled": notificationcommon.EnabledAttribute("Telegram"),
		"token":   sensStr("Telegram bot token."),
		"chat_id": sensStr("Telegram chat ID."),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the current team's Telegram notification settings in Coolify. " +
			"This is a team-scoped singleton (one configuration per team, selected by the API token). " +
			"**Requires Coolify >= v4.3.0** (notification routes are absent on v4.2.x and older; acceptance tests skip when the API is missing).\n\n" +
			"On destroy, Telegram notifications are disabled (`enabled = false`); token, chat ID, and thread IDs are left unchanged. " +
			"Import with id `current`.",
		Attributes: notificationcommon.MergeAttrs(
			notificationcommon.MergeAttrs(attrs, notificationcommon.EventSchemaAttrs("Telegram")),
			notificationcommon.ThreadSchemaAttrs("Telegram"),
		),
	}
}

func (r *telegramResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *telegramResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_notification_telegram"})
	input, err := createInputFromPlan(plan)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	updated, err := r.client.UpdateTelegramNotifications(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error configuring Telegram notifications", err.Error())
		return
	}
	if err := flatten(updated, &plan); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *telegramResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_notification_telegram"})
	got, err := r.client.GetTelegramNotifications(ctx)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Telegram notifications", err.Error())
		return
	}
	if err := flatten(got, &state); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *telegramResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_notification_telegram"})
	input, err := updateInputFromPlan(plan, state)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	updated, err := r.client.UpdateTelegramNotifications(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Telegram notifications", err.Error())
		return
	}
	if err := flatten(updated, &plan); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *telegramResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "deleting resource (disabling Telegram notifications)", map[string]interface{}{"resource_type": "coolify_notification_telegram"})
	enabled := false
	_, err := r.client.UpdateTelegramNotifications(ctx, client.UpdateTelegramNotificationInput{
		Enabled: &enabled,
	})
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error disabling Telegram notifications on destroy", err.Error())
	}
}

func (r *telegramResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	notificationcommon.ImportStateCurrent(ctx, req, resp, "coolify_notification_telegram")
}

func createInputFromPlan(plan model) (client.UpdateTelegramNotificationInput, error) {
	in := client.UpdateTelegramNotificationInput{
		Enabled: flex.BoolValueOrNull(plan.Enabled),
	}
	if err := client.ApplyEventUpdate(&in, plan.CreateUpdate()); err != nil {
		return in, err
	}
	if err := client.ApplyThreadUpdate(&in, plan.CreateThreadUpdate()); err != nil {
		return in, err
	}
	if flex.StringValueConfigured(plan.Token) {
		s := plan.Token.ValueString()
		in.Token = &s
	}
	if flex.StringValueConfigured(plan.ChatID) {
		s := plan.ChatID.ValueString()
		in.ChatID = &s
	}
	return in, nil
}

func updateInputFromPlan(plan, state model) (client.UpdateTelegramNotificationInput, error) {
	in := client.UpdateTelegramNotificationInput{
		Enabled: flex.BoolIfChanged(plan.Enabled, state.Enabled),
	}
	if err := client.ApplyEventUpdate(&in, plan.DiffUpdate(state.EventModel)); err != nil {
		return in, err
	}
	if err := client.ApplyThreadUpdate(&in, plan.DiffThreadUpdate(state.ThreadModel)); err != nil {
		return in, err
	}
	if w := flex.StringIfChanged(plan.Token, state.Token); w != nil {
		in.Token = w
	}
	if w := flex.StringIfChanged(plan.ChatID, state.ChatID); w != nil {
		in.ChatID = w
	}
	return in, nil
}

func flatten(api *client.TelegramNotificationSettings, m *model) error {
	ev, err := client.EventsFrom(api)
	if err != nil {
		return err
	}
	m.FlattenEvents(ev)
	th, err := client.ThreadsFrom(api)
	if err != nil {
		return err
	}
	m.FlattenThreads(th)
	m.ID = types.StringValue(notificationcommon.ImportIDCurrent)
	m.Enabled = types.BoolValue(api.Enabled)
	flex.SetStringPreserveEmpty(&m.Token, api.Token)
	flex.SetStringPreserveEmpty(&m.ChatID, api.ChatID)
	return nil
}

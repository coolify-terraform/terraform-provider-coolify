package notificationpushover

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
	_ resource.Resource                = (*pushoverResource)(nil)
	_ resource.ResourceWithConfigure   = (*pushoverResource)(nil)
	_ resource.ResourceWithImportState = (*pushoverResource)(nil)
)

type pushoverResource struct {
	client *client.Client
}

type model struct {
	ID       types.String `tfsdk:"id"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	UserKey  types.String `tfsdk:"user_key"`
	APIToken types.String `tfsdk:"api_token"`

	notificationcommon.EventModel
}

// NewResource returns a new Pushover notification resource.
func NewResource() resource.Resource {
	return &pushoverResource{}
}

func (r *pushoverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_pushover"
}

func (r *pushoverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id":      notificationcommon.IDAttribute(),
		"enabled": notificationcommon.EnabledAttribute("Pushover"),
		"user_key": schema.StringAttribute{
			MarkdownDescription: "Pushover user key. Sensitive; Coolify may omit it on read unless the API token " +
				"can read sensitive fields (`read:sensitive` or root). Preserve the value in configuration after import.",
			Optional:  true,
			Computed:  true,
			Sensitive: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"api_token": schema.StringAttribute{
			MarkdownDescription: "Pushover API token. Sensitive; Coolify may omit it on read unless the API token " +
				"can read sensitive fields (`read:sensitive` or root). Preserve the value in configuration after import.",
			Optional:  true,
			Computed:  true,
			Sensitive: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the current team's Pushover notification settings in Coolify. " +
			"This is a team-scoped singleton (one configuration per team, selected by the API token). " +
			"Requires Coolify >= v4.3.0.\n\n" +
			"On destroy, Pushover notifications are disabled (`enabled = false`); credentials are left unchanged. " +
			"Import with id `current`.",
		Attributes: notificationcommon.MergeAttrs(attrs, notificationcommon.EventSchemaAttrs("Pushover")),
	}
}

func (r *pushoverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *pushoverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_notification_pushover"})

	input, err := createInputFromPlan(plan)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping notification events", err.Error())
		return
	}
	updated, err := r.client.UpdatePushoverNotifications(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error configuring Pushover notifications", err.Error())
		return
	}

	if err := flatten(updated, &plan); err != nil {
		resp.Diagnostics.AddError("Error mapping notification events", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pushoverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_notification_pushover"})

	got, err := r.client.GetPushoverNotifications(ctx)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Pushover notifications", err.Error())
		return
	}

	if err := flatten(got, &state); err != nil {
		resp.Diagnostics.AddError("Error mapping notification events", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pushoverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_notification_pushover"})

	input, err := updateInputFromPlan(plan, state)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping notification events", err.Error())
		return
	}
	updated, err := r.client.UpdatePushoverNotifications(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Pushover notifications", err.Error())
		return
	}

	if err := flatten(updated, &plan); err != nil {
		resp.Diagnostics.AddError("Error mapping notification events", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pushoverResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "deleting resource (disabling Pushover notifications)", map[string]interface{}{"resource_type": "coolify_notification_pushover"})
	enabled := false
	_, err := r.client.UpdatePushoverNotifications(ctx, client.UpdatePushoverNotificationInput{
		Enabled: &enabled,
	})
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error disabling Pushover notifications on destroy", err.Error())
	}
}

func (r *pushoverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	notificationcommon.ImportStateCurrent(ctx, req, resp, "coolify_notification_pushover")
}

func createInputFromPlan(plan model) (client.UpdatePushoverNotificationInput, error) {
	in := client.UpdatePushoverNotificationInput{
		Enabled: flex.BoolValueOrNull(plan.Enabled),
	}
	if err := client.ApplyEventUpdate(&in, plan.CreateUpdate()); err != nil {
		return in, err
	}
	if flex.StringValueConfigured(plan.UserKey) {
		v := plan.UserKey.ValueString()
		in.UserKey = &v
	}
	if flex.StringValueConfigured(plan.APIToken) {
		v := plan.APIToken.ValueString()
		in.APIToken = &v
	}
	return in, nil
}

func updateInputFromPlan(plan, state model) (client.UpdatePushoverNotificationInput, error) {
	in := client.UpdatePushoverNotificationInput{
		Enabled: flex.BoolIfChanged(plan.Enabled, state.Enabled),
	}
	if err := client.ApplyEventUpdate(&in, plan.DiffUpdate(state.EventModel)); err != nil {
		return in, err
	}
	if w := flex.StringIfChanged(plan.UserKey, state.UserKey); w != nil {
		in.UserKey = w
	}
	if w := flex.StringIfChanged(plan.APIToken, state.APIToken); w != nil {
		in.APIToken = w
	}
	return in, nil
}

func flatten(api *client.PushoverNotificationSettings, m *model) error {
	ev, err := client.EventsFrom(api)
	if err != nil {
		return err
	}
	m.FlattenEvents(ev)
	m.ID = types.StringValue(notificationcommon.ImportIDCurrent)
	m.Enabled = types.BoolValue(api.Enabled)
	flex.SetStringPreserveEmpty(&m.UserKey, api.UserKey)
	flex.SetStringPreserveEmpty(&m.APIToken, api.APIToken)
	return nil
}

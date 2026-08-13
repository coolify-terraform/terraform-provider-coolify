package notificationemail

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*emailResource)(nil)
	_ resource.ResourceWithConfigure   = (*emailResource)(nil)
	_ resource.ResourceWithImportState = (*emailResource)(nil)
)

type emailResource struct {
	client *client.Client
}

type model struct {
	ID types.String `tfsdk:"id"`

	SMTPEnabled      types.Bool   `tfsdk:"smtp_enabled"`
	SMTPFromAddress  types.String `tfsdk:"smtp_from_address"`
	SMTPFromName     types.String `tfsdk:"smtp_from_name"`
	SMTPRecipients   types.String `tfsdk:"smtp_recipients"`
	SMTPHost         types.String `tfsdk:"smtp_host"`
	SMTPPort         types.Int64  `tfsdk:"smtp_port"`
	SMTPEncryption   types.String `tfsdk:"smtp_encryption"`
	SMTPUsername     types.String `tfsdk:"smtp_username"`
	SMTPPassword     types.String `tfsdk:"smtp_password"`
	SMTPTimeout      types.Int64  `tfsdk:"smtp_timeout"`
	ResendEnabled    types.Bool   `tfsdk:"resend_enabled"`
	ResendAPIKey     types.String `tfsdk:"resend_api_key"`
	UseInstanceEmail types.Bool   `tfsdk:"use_instance_email_settings"`

	notificationcommon.EventModel
}

func NewResource() resource.Resource {
	return &emailResource{}
}

func (r *emailResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_email"
}

func (r *emailResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
		"id": notificationcommon.IDAttribute(),

		"smtp_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether SMTP delivery is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"smtp_from_address": sensStr("SMTP From address."),
		"smtp_from_name":    sensStr("SMTP From display name."),
		"smtp_recipients":   sensStr("Comma-separated recipient addresses."),
		"smtp_host":         sensStr("SMTP host."),
		"smtp_port": schema.Int64Attribute{
			MarkdownDescription: "SMTP port (1-65535).",
			Optional:            true,
			Computed:            true,
			Validators: []validator.Int64{
				int64validator.Between(1, 65535),
			},
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"smtp_encryption": schema.StringAttribute{
			MarkdownDescription: "SMTP encryption. One of `starttls`, `tls`, or `none`.",
			Optional:            true,
			Computed:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("starttls", "tls", "none"),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"smtp_username": sensStr("SMTP username."),
		"smtp_password": sensStr("SMTP password."),
		"smtp_timeout": schema.Int64Attribute{
			MarkdownDescription: "SMTP timeout in seconds (>= 0).",
			Optional:            true,
			Computed:            true,
			Validators: []validator.Int64{
				int64validator.AtLeast(0),
			},
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"resend_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether Resend delivery is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"resend_api_key": sensStr("Resend API key."),
		"use_instance_email_settings": schema.BoolAttribute{
			MarkdownDescription: "Whether to use the Coolify instance's shared email settings for this team.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the current team's email notification settings in Coolify. " +
			"This is a team-scoped singleton (one configuration per team, selected by the API token). " +
			"**Requires Coolify >= v4.3.0** (notification routes are absent on v4.2.x and older; acceptance tests skip when the API is missing).\n\n" +
			"Email can use SMTP, Resend, and/or instance-level settings. Coolify treats the channel as enabled when any of " +
			"`smtp_enabled`, `resend_enabled`, or `use_instance_email_settings` is true.\n\n" +
			"On destroy, all three enable flags are set to `false`; credentials are left unchanged. Import with id `current`.",
		Attributes: notificationcommon.MergeAttrs(attrs, notificationcommon.EventSchemaAttrs("email")),
	}
}

func (r *emailResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *emailResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_notification_email"})
	updated, err := r.client.UpdateEmailNotifications(ctx, createInputFromPlan(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error configuring email notifications", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *emailResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_notification_email"})
	got, err := r.client.GetEmailNotifications(ctx)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading email notifications", err.Error())
		return
	}
	flatten(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *emailResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_notification_email"})
	updated, err := r.client.UpdateEmailNotifications(ctx, updateInputFromPlan(plan, state))
	if err != nil {
		resp.Diagnostics.AddError("Error updating email notifications", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *emailResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "deleting resource (disabling email notifications)", map[string]interface{}{"resource_type": "coolify_notification_email"})
	f := false
	_, err := r.client.UpdateEmailNotifications(ctx, client.UpdateEmailNotificationInput{
		SMTPEnabled:      &f,
		ResendEnabled:    &f,
		UseInstanceEmail: &f,
	})
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error disabling email notifications on destroy", err.Error())
	}
}

func (r *emailResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	notificationcommon.ImportStateCurrent(ctx, req, resp, "coolify_notification_email")
}

func createInputFromPlan(plan model) client.UpdateEmailNotificationInput {
	in := client.UpdateEmailNotificationInput{
		SMTPEnabled:      flex.BoolValueOrNull(plan.SMTPEnabled),
		ResendEnabled:    flex.BoolValueOrNull(plan.ResendEnabled),
		UseInstanceEmail: flex.BoolValueOrNull(plan.UseInstanceEmail),
	}
	_ = client.ApplyEventUpdate(&in, plan.CreateUpdate())
	setStr := func(v types.String, dst **string) {
		if flex.StringValueConfigured(v) {
			s := v.ValueString()
			*dst = &s
		}
	}
	setStr(plan.SMTPFromAddress, &in.SMTPFromAddress)
	setStr(plan.SMTPFromName, &in.SMTPFromName)
	setStr(plan.SMTPRecipients, &in.SMTPRecipients)
	setStr(plan.SMTPHost, &in.SMTPHost)
	setStr(plan.SMTPEncryption, &in.SMTPEncryption)
	setStr(plan.SMTPUsername, &in.SMTPUsername)
	setStr(plan.SMTPPassword, &in.SMTPPassword)
	setStr(plan.ResendAPIKey, &in.ResendAPIKey)
	if !plan.SMTPPort.IsNull() && !plan.SMTPPort.IsUnknown() {
		v := int(plan.SMTPPort.ValueInt64())
		in.SMTPPort = &v
	}
	if !plan.SMTPTimeout.IsNull() && !plan.SMTPTimeout.IsUnknown() {
		v := int(plan.SMTPTimeout.ValueInt64())
		in.SMTPTimeout = &v
	}
	return in
}

func updateInputFromPlan(plan, state model) client.UpdateEmailNotificationInput {
	in := client.UpdateEmailNotificationInput{
		SMTPEnabled:      flex.BoolIfChanged(plan.SMTPEnabled, state.SMTPEnabled),
		ResendEnabled:    flex.BoolIfChanged(plan.ResendEnabled, state.ResendEnabled),
		UseInstanceEmail: flex.BoolIfChanged(plan.UseInstanceEmail, state.UseInstanceEmail),
	}
	_ = client.ApplyEventUpdate(&in, plan.DiffUpdate(state.EventModel))
	if w := flex.StringIfChanged(plan.SMTPFromAddress, state.SMTPFromAddress); w != nil {
		in.SMTPFromAddress = w
	}
	if w := flex.StringIfChanged(plan.SMTPFromName, state.SMTPFromName); w != nil {
		in.SMTPFromName = w
	}
	if w := flex.StringIfChanged(plan.SMTPRecipients, state.SMTPRecipients); w != nil {
		in.SMTPRecipients = w
	}
	if w := flex.StringIfChanged(plan.SMTPHost, state.SMTPHost); w != nil {
		in.SMTPHost = w
	}
	if w := flex.StringIfChanged(plan.SMTPEncryption, state.SMTPEncryption); w != nil {
		in.SMTPEncryption = w
	}
	if w := flex.StringIfChanged(plan.SMTPUsername, state.SMTPUsername); w != nil {
		in.SMTPUsername = w
	}
	if w := flex.StringIfChanged(plan.SMTPPassword, state.SMTPPassword); w != nil {
		in.SMTPPassword = w
	}
	if w := flex.StringIfChanged(plan.ResendAPIKey, state.ResendAPIKey); w != nil {
		in.ResendAPIKey = w
	}
	in.SMTPPort = flex.IntIfChanged(plan.SMTPPort, state.SMTPPort)
	in.SMTPTimeout = flex.IntIfChanged(plan.SMTPTimeout, state.SMTPTimeout)
	return in
}

func flatten(api *client.EmailNotificationSettings, m *model) {
	if ev, err := client.EventsFrom(api); err == nil {
		m.FlattenEvents(ev)
	}
	m.ID = types.StringValue(notificationcommon.ImportIDCurrent)
	m.SMTPEnabled = types.BoolValue(api.SMTPEnabled)
	m.ResendEnabled = types.BoolValue(api.ResendEnabled)
	m.UseInstanceEmail = types.BoolValue(api.UseInstanceEmail)
	flex.SetStringPreserveEmpty(&m.SMTPFromAddress, api.SMTPFromAddress)
	flex.SetStringPreserveEmpty(&m.SMTPFromName, api.SMTPFromName)
	flex.SetStringPreserveEmpty(&m.SMTPRecipients, api.SMTPRecipients)
	flex.SetStringPreserveEmpty(&m.SMTPHost, api.SMTPHost)
	m.SMTPEncryption = types.StringValue(api.SMTPEncryption)
	if api.SMTPEncryption == "" {
		m.SMTPEncryption = types.StringNull()
	}
	flex.SetStringPreserveEmpty(&m.SMTPUsername, api.SMTPUsername)
	flex.SetStringPreserveEmpty(&m.SMTPPassword, api.SMTPPassword)
	flex.SetStringPreserveEmpty(&m.ResendAPIKey, api.ResendAPIKey)
	if api.SMTPPort != nil {
		m.SMTPPort = types.Int64Value(int64(*api.SMTPPort))
	} else if m.SMTPPort.IsNull() || m.SMTPPort.IsUnknown() {
		m.SMTPPort = types.Int64Null()
	}
	if api.SMTPTimeout != nil {
		m.SMTPTimeout = types.Int64Value(int64(*api.SMTPTimeout))
	} else if m.SMTPTimeout.IsNull() || m.SMTPTimeout.IsUnknown() {
		m.SMTPTimeout = types.Int64Null()
	}
}

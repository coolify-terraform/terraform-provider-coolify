package instanceemail

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = (*instanceEmailResource)(nil)
	_ resource.ResourceWithConfigure   = (*instanceEmailResource)(nil)
	_ resource.ResourceWithImportState = (*instanceEmailResource)(nil)
)

type instanceEmailResource struct {
	client *client.Client
}

type model struct {
	ID              types.String `tfsdk:"id"`
	SMTPEnabled     types.Bool   `tfsdk:"smtp_enabled"`
	SMTPFromAddress types.String `tfsdk:"smtp_from_address"`
	SMTPFromName    types.String `tfsdk:"smtp_from_name"`
	SMTPHost        types.String `tfsdk:"smtp_host"`
	SMTPPort        types.Int64  `tfsdk:"smtp_port"`
	SMTPEncryption  types.String `tfsdk:"smtp_encryption"`
	SMTPUsername    types.String `tfsdk:"smtp_username"`
	SMTPPassword    types.String `tfsdk:"smtp_password"`
	SMTPTimeout     types.Int64  `tfsdk:"smtp_timeout"`
	SMTPEhloDomain  types.String `tfsdk:"smtp_ehlo_domain"`
	ResendEnabled   types.Bool   `tfsdk:"resend_enabled"`
	ResendAPIKey    types.String `tfsdk:"resend_api_key"`
}

func NewResource() resource.Resource {
	return &instanceEmailResource{}
}

func (r *instanceEmailResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_email_settings"
}

func (r *instanceEmailResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Coolify instance-wide SMTP and Resend settings (`GET`/`PATCH /settings/email`). " +
			"This is an instance singleton. It requires a **root-team** API token (team 0) belonging to a root-team admin or owner. " +
			"**Requires Coolify >= v4.3.10** (the route is absent on v4.3.9 and older; acceptance tests skip when the API is missing).\n\n" +
			"Team notification email (`coolify_notification_email`) can inherit these values with `use_instance_email_settings = true`.\n\n" +
			"On destroy, `smtp_enabled` and `resend_enabled` are set to `false`; credentials are left unchanged. Import with id `current`.",
		Attributes: map[string]schema.Attribute{
			"id": notificationcommon.IDAttribute(),
			"smtp_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether instance SMTP delivery is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"smtp_from_address": notificationcommon.StringSensitiveOmit("SMTP From address."),
			"smtp_from_name":    notificationcommon.StringSensitiveOmit("SMTP From display name."),
			"smtp_host":         notificationcommon.StringSensitiveOmit("SMTP host."),
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
			"smtp_username": notificationcommon.StringSensitiveOmit("SMTP username."),
			"smtp_password": notificationcommon.StringSensitiveOmit("SMTP password."),
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
			"smtp_ehlo_domain": schema.StringAttribute{
				MarkdownDescription: "Hostname sent with SMTP EHLO. Set a valid hostname, or omit to use Coolify's default. " +
					"Requires Coolify >= v4.3.10 ([coollabsio/coolify#11398](https://github.com/coollabsio/coolify/pull/11398)).",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(253),
					validate.Hostname(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resend_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether instance Resend delivery is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"resend_api_key": notificationcommon.StringSensitiveOmit("Resend API key."),
		},
	}
}

func (r *instanceEmailResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *instanceEmailResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_instance_email_settings"})
	if !r.client.SupportsInstanceEmailSettings() {
		warnUnsupportedInstanceEmail(r.client, &resp.Diagnostics)
		resp.Diagnostics.AddError(
			"Coolify version cannot manage instance email settings",
			fmt.Sprintf("This Coolify instance (%s) has no GET/PATCH /settings/email (added in v4.3.10). Upgrade Coolify, or remove this resource.", versionOrUnknown(r.client)),
		)
		return
	}
	updated, err := r.client.UpdateInstanceEmailSettings(ctx, createInputFromPlan(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error configuring instance email settings", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceEmailResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_instance_email_settings"})
	got, err := r.client.GetInstanceEmailSettings(ctx)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading instance email settings", err.Error())
		return
	}
	flatten(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *instanceEmailResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_instance_email_settings"})
	updated, err := r.client.UpdateInstanceEmailSettings(ctx, updateInputFromPlan(plan, state))
	if err != nil {
		resp.Diagnostics.AddError("Error updating instance email settings", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceEmailResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "deleting resource (disabling instance email)", map[string]interface{}{"resource_type": "coolify_instance_email_settings"})
	f := false
	_, err := r.client.UpdateInstanceEmailSettings(ctx, client.UpdateInstanceEmailInput{
		SMTPEnabled:   &f,
		ResendEnabled: &f,
	})
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error disabling instance email settings on destroy", err.Error())
	}
}

func (r *instanceEmailResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	notificationcommon.ImportStateCurrent(ctx, req, resp, "coolify_instance_email_settings")
}

func createInputFromPlan(plan model) client.UpdateInstanceEmailInput {
	in := client.UpdateInstanceEmailInput{
		SMTPEnabled:   flex.BoolValueOrNull(plan.SMTPEnabled),
		ResendEnabled: flex.BoolValueOrNull(plan.ResendEnabled),
	}
	setStr := func(v types.String, dst **string) {
		if flex.StringValueConfigured(v) {
			s := v.ValueString()
			*dst = &s
		}
	}
	setStr(plan.SMTPFromAddress, &in.SMTPFromAddress)
	setStr(plan.SMTPFromName, &in.SMTPFromName)
	setStr(plan.SMTPHost, &in.SMTPHost)
	setStr(plan.SMTPEncryption, &in.SMTPEncryption)
	setStr(plan.SMTPUsername, &in.SMTPUsername)
	setStr(plan.SMTPPassword, &in.SMTPPassword)
	setStr(plan.SMTPEhloDomain, &in.SMTPEhloDomain)
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

func updateInputFromPlan(plan, state model) client.UpdateInstanceEmailInput {
	in := client.UpdateInstanceEmailInput{
		SMTPEnabled:   flex.BoolIfChanged(plan.SMTPEnabled, state.SMTPEnabled),
		ResendEnabled: flex.BoolIfChanged(plan.ResendEnabled, state.ResendEnabled),
	}
	if w := flex.StringIfChanged(plan.SMTPFromAddress, state.SMTPFromAddress); w != nil {
		in.SMTPFromAddress = w
	}
	if w := flex.StringIfChanged(plan.SMTPFromName, state.SMTPFromName); w != nil {
		in.SMTPFromName = w
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
	if w := flex.StringIfChanged(plan.SMTPEhloDomain, state.SMTPEhloDomain); w != nil {
		in.SMTPEhloDomain = w
	}
	if w := flex.StringIfChanged(plan.ResendAPIKey, state.ResendAPIKey); w != nil {
		in.ResendAPIKey = w
	}
	in.SMTPPort = flex.IntIfChanged(plan.SMTPPort, state.SMTPPort)
	in.SMTPTimeout = flex.IntIfChanged(plan.SMTPTimeout, state.SMTPTimeout)
	return in
}

func flatten(api *client.InstanceEmailSettings, m *model) {
	m.ID = types.StringValue(notificationcommon.ImportIDCurrent)
	m.SMTPEnabled = types.BoolValue(api.SMTPEnabled)
	m.ResendEnabled = types.BoolValue(api.ResendEnabled)
	flex.SetStringPreserveEmpty(&m.SMTPFromAddress, api.SMTPFromAddress)
	flex.SetStringPreserveEmpty(&m.SMTPFromName, api.SMTPFromName)
	flex.SetStringPreserveEmpty(&m.SMTPHost, api.SMTPHost)
	m.SMTPEncryption = types.StringValue(api.SMTPEncryption)
	if api.SMTPEncryption == "" {
		m.SMTPEncryption = types.StringNull()
	}
	flex.SetStringPreserveEmpty(&m.SMTPUsername, api.SMTPUsername)
	flex.SetStringPreserveEmpty(&m.SMTPPassword, api.SMTPPassword)
	ehlo := ""
	if api.SMTPEhloDomain != nil {
		ehlo = *api.SMTPEhloDomain
	}
	flex.SetStringPreserveEmpty(&m.SMTPEhloDomain, ehlo)
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

func warnUnsupportedInstanceEmail(c *client.Client, diags *diag.Diagnostics) {
	if diags == nil {
		return
	}
	diags.AddWarning(
		"Coolify version cannot manage instance email settings",
		fmt.Sprintf("This Coolify instance (%s) has no GET/PATCH /settings/email (added in v4.3.10).", versionOrUnknown(c)),
	)
}

func versionOrUnknown(c *client.Client) string {
	if c == nil || c.CoolifyVersion == "" {
		return "unknown"
	}
	return c.CoolifyVersion
}

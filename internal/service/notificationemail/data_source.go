package notificationemail

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*emailDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*emailDataSource)(nil)
)

type emailDataSource struct {
	client *client.Client
}

// NewDataSource returns the team email notification data source.
func NewDataSource() datasource.DataSource {
	return &emailDataSource{}
}

func (d *emailDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_email"
}

func (d *emailDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	sens := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: desc + " Coolify may omit this unless the API token can read sensitive fields (`read:sensitive` or root).",
			Computed:            true,
			Sensitive:           true,
		}
	}
	attrs := map[string]schema.Attribute{
		"id":                          notificationcommon.IDAttributeDS(),
		"smtp_enabled":                notificationcommon.BoolComputed("Whether SMTP delivery is enabled."),
		"smtp_from_address":           sens("SMTP From address."),
		"smtp_from_name":              sens("SMTP From display name."),
		"smtp_recipients":             sens("Comma-separated recipient addresses."),
		"smtp_host":                   sens("SMTP host."),
		"smtp_port":                   schema.Int64Attribute{MarkdownDescription: "SMTP port.", Computed: true},
		"smtp_encryption":             schema.StringAttribute{MarkdownDescription: "SMTP encryption (`starttls`, `tls`, or `none`).", Computed: true},
		"smtp_username":               sens("SMTP username."),
		"smtp_password":               sens("SMTP password."),
		"smtp_timeout":                schema.Int64Attribute{MarkdownDescription: "SMTP timeout in seconds.", Computed: true},
		"resend_enabled":              notificationcommon.BoolComputed("Whether Resend delivery is enabled."),
		"resend_api_key":              sens("Resend API key."),
		"use_instance_email_settings": notificationcommon.BoolComputed("Whether the team uses the Coolify instance email settings."),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the current team's email notification settings (SMTP, Resend, and/or instance settings). " +
			"This is a team-scoped singleton (selected by the API token). Requires Coolify >= v4.3.0.",
		Attributes: notificationcommon.MergeDSAttrs(attrs, notificationcommon.EventDataSourceAttrs("email")),
	}
}

func (d *emailDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *emailDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_notification_email"})
	got, err := d.client.GetEmailNotifications(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading email notifications", err.Error())
		return
	}
	var state model
	if err := flatten(got, &state); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

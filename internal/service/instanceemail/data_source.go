package instanceemail

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*instanceEmailDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*instanceEmailDataSource)(nil)
)

type instanceEmailDataSource struct {
	client *client.Client
}

// NewDataSource returns the instance-wide SMTP/Resend settings data source.
func NewDataSource() datasource.DataSource {
	return &instanceEmailDataSource{}
}

func (d *instanceEmailDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_email_settings"
}

func (d *instanceEmailDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	sens := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: desc + " Coolify may omit this unless the API token can read sensitive fields (`read:sensitive` or root).",
			Computed:            true,
			Sensitive:           true,
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Coolify instance-wide SMTP and Resend settings (`GET /settings/email`). " +
			"This is an instance singleton. It requires a **root-team** API token (team 0). " +
			"**Requires Coolify >= v4.3.10** (the route is absent on v4.3.9 and older; acceptance tests skip when the API is missing).",
		Attributes: map[string]schema.Attribute{
			"id":                notificationcommon.IDAttributeDS(),
			"smtp_enabled":      notificationcommon.BoolComputed("Whether instance SMTP delivery is enabled."),
			"smtp_from_address": sens("SMTP From address."),
			"smtp_from_name":    sens("SMTP From display name."),
			"smtp_host":         sens("SMTP host."),
			"smtp_port":         schema.Int64Attribute{MarkdownDescription: "SMTP port.", Computed: true},
			"smtp_encryption":   schema.StringAttribute{MarkdownDescription: "SMTP encryption (`starttls`, `tls`, or `none`).", Computed: true},
			"smtp_username":     sens("SMTP username."),
			"smtp_password":     sens("SMTP password."),
			"smtp_timeout":      schema.Int64Attribute{MarkdownDescription: "SMTP timeout in seconds.", Computed: true},
			"smtp_ehlo_domain": schema.StringAttribute{
				MarkdownDescription: "Hostname sent with SMTP EHLO. Requires Coolify >= v4.3.10.",
				Computed:            true,
			},
			"resend_enabled": notificationcommon.BoolComputed("Whether instance Resend delivery is enabled."),
			"resend_api_key": sens("Resend API key."),
		},
	}
}

func (d *instanceEmailDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *instanceEmailDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_instance_email_settings"})
	if !d.client.SupportsInstanceEmailSettings() {
		warnUnsupportedInstanceEmail(d.client, &resp.Diagnostics)
		resp.Diagnostics.AddError(
			"Coolify version cannot read instance email settings",
			fmt.Sprintf("This Coolify instance (%s) has no GET /settings/email (added in v4.3.10). Upgrade Coolify, or remove this data source.", versionOrUnknown(d.client)),
		)
		return
	}
	got, err := d.client.GetInstanceEmailSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading instance email settings", err.Error())
		return
	}
	var state model
	flatten(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

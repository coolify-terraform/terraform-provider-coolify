package notificationwebhook

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
	_ datasource.DataSource              = (*webhookDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*webhookDataSource)(nil)
)

type webhookDataSource struct {
	client *client.Client
}

// NewDataSource returns the team webhook notification data source.
func NewDataSource() datasource.DataSource {
	return &webhookDataSource{}
}

func (d *webhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_webhook"
}

func (d *webhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id":      notificationcommon.IDAttributeDS(),
		"enabled": notificationcommon.EnabledAttributeDS("Webhook"),
		"webhook_url": schema.StringAttribute{
			MarkdownDescription: "Generic webhook URL. Coolify may omit this unless the API token can read sensitive fields (`read:sensitive` or root).",
			Computed:            true,
			Sensitive:           true,
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the current team's generic webhook notification settings. " +
			"This is a team-scoped singleton (selected by the API token). Requires Coolify >= v4.3.0.",
		Attributes: notificationcommon.MergeDSAttrs(attrs, notificationcommon.EventDataSourceAttrs("Webhook")),
	}
}

func (d *webhookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *webhookDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_notification_webhook"})
	got, err := d.client.GetWebhookNotifications(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Webhook notifications", err.Error())
		return
	}
	var state model
	if err := flatten(got, &state); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

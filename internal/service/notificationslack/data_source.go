package notificationslack

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
	_ datasource.DataSource              = (*slackDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*slackDataSource)(nil)
)

type slackDataSource struct {
	client *client.Client
}

// NewDataSource returns the team Slack notification data source.
func NewDataSource() datasource.DataSource {
	return &slackDataSource{}
}

func (d *slackDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_slack"
}

func (d *slackDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id":      notificationcommon.IDAttributeDS(),
		"enabled": notificationcommon.EnabledAttributeDS("Slack"),
		"webhook_url": schema.StringAttribute{
			MarkdownDescription: "Slack incoming webhook URL. Coolify may omit this unless the API token can read sensitive fields (`read:sensitive` or root).",
			Computed:            true,
			Sensitive:           true,
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the current team's Slack notification settings. " +
			"This is a team-scoped singleton (selected by the API token). Requires Coolify >= v4.3.0.",
		Attributes: notificationcommon.MergeDSAttrs(attrs, notificationcommon.EventDataSourceAttrs("Slack")),
	}
}

func (d *slackDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *slackDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_notification_slack"})
	got, err := d.client.GetSlackNotifications(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Slack notifications", err.Error())
		return
	}
	var state model
	if err := flatten(got, &state); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

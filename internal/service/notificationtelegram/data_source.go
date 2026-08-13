package notificationtelegram

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
	_ datasource.DataSource              = (*telegramDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*telegramDataSource)(nil)
)

type telegramDataSource struct {
	client *client.Client
}

// NewDataSource returns the team Telegram notification data source.
func NewDataSource() datasource.DataSource {
	return &telegramDataSource{}
}

func (d *telegramDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_telegram"
}

func (d *telegramDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	sens := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: desc + " Coolify may omit this unless the API token can read sensitive fields (`read:sensitive` or root).",
			Computed:            true,
			Sensitive:           true,
		}
	}
	attrs := map[string]schema.Attribute{
		"id":      notificationcommon.IDAttributeDS(),
		"enabled": notificationcommon.EnabledAttributeDS("Telegram"),
		"token":   sens("Telegram bot token."),
		"chat_id": sens("Telegram chat ID."),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the current team's Telegram notification settings. " +
			"This is a team-scoped singleton (selected by the API token). Requires Coolify >= v4.3.0.",
		Attributes: notificationcommon.MergeDSAttrs(attrs, notificationcommon.EventDataSourceAttrs("Telegram"), notificationcommon.ThreadDataSourceAttrs()),
	}
}

func (d *telegramDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *telegramDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_notification_telegram"})
	got, err := d.client.GetTelegramNotifications(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Telegram notifications", err.Error())
		return
	}
	var state model
	if err := flatten(got, &state); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

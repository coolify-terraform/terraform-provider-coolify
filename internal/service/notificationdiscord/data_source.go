package notificationdiscord

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
	_ datasource.DataSource              = (*discordDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*discordDataSource)(nil)
)

type discordDataSource struct {
	client *client.Client
}

// NewDataSource returns the team Discord notification data source.
func NewDataSource() datasource.DataSource {
	return &discordDataSource{}
}

func (d *discordDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_discord"
}

func (d *discordDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id":      notificationcommon.IDAttributeDS(),
		"enabled": notificationcommon.EnabledAttributeDS("Discord"),
		"webhook_url": schema.StringAttribute{
			MarkdownDescription: "Discord webhook URL. Coolify may omit this unless the API token can read sensitive fields (`read:sensitive` or root).",
			Computed:            true,
			Sensitive:           true,
		},
		"ping_enabled": notificationcommon.BoolComputed("Whether Discord @here / role pings are enabled."),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the current team's Discord notification settings. " +
			"This is a team-scoped singleton (selected by the API token). Requires Coolify >= v4.3.0.",
		Attributes: notificationcommon.MergeDSAttrs(attrs, notificationcommon.EventDataSourceAttrs("Discord")),
	}
}

func (d *discordDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *discordDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_notification_discord"})
	got, err := d.client.GetDiscordNotifications(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Discord notifications", err.Error())
		return
	}
	var state model
	if err := flatten(got, &state); err != nil {
		resp.Diagnostics.AddError("Error mapping notification settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

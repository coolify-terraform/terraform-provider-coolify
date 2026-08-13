package notificationpushover

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
	_ datasource.DataSource              = (*pushoverDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*pushoverDataSource)(nil)
)

type pushoverDataSource struct {
	client *client.Client
}

// NewDataSource returns the team Pushover notification data source.
func NewDataSource() datasource.DataSource {
	return &pushoverDataSource{}
}

func (d *pushoverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_pushover"
}

func (d *pushoverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	sens := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: desc + " Coolify may omit this unless the API token can read sensitive fields (`read:sensitive` or root).",
			Computed:            true,
			Sensitive:           true,
		}
	}
	attrs := map[string]schema.Attribute{
		"id":        notificationcommon.IDAttributeDS(),
		"enabled":   notificationcommon.EnabledAttributeDS("Pushover"),
		"user_key":  sens("Pushover user key."),
		"api_token": sens("Pushover API token."),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the current team's Pushover notification settings. " +
			"This is a team-scoped singleton (selected by the API token). Requires Coolify >= v4.3.0.",
		Attributes: notificationcommon.MergeDSAttrs(attrs, notificationcommon.EventDataSourceAttrs("Pushover")),
	}
}

func (d *pushoverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *pushoverDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_notification_pushover"})
	got, err := d.client.GetPushoverNotifications(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Pushover notifications", err.Error())
		return
	}
	var state model
	if err := flatten(got, &state); err != nil {
		resp.Diagnostics.AddError("Error mapping notification events", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

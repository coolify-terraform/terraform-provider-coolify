package githubapp

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*gitHubAppDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*gitHubAppDataSource)(nil)
)

type gitHubAppDataSource struct {
	client *client.Client
}

type gitHubAppDataSourceModel struct {
	ID               types.Int64  `tfsdk:"id"`
	UUID             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	OrganizationName types.String `tfsdk:"organization_name"`
	AppID            types.Int64  `tfsdk:"app_id"`
	InstallationID   types.Int64  `tfsdk:"installation_id"`
	ClientID         types.String `tfsdk:"client_id"`
	WebhookSecret    types.String `tfsdk:"webhook_secret"`
}

// NewDataSource returns a new singular GitHub App data source.
func NewDataSource() datasource.DataSource {
	return &gitHubAppDataSource{}
}

func (d *gitHubAppDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_app"
}

func (d *gitHubAppDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Coolify GitHub App integration by its numeric ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The numeric identifier of the GitHub App.",
				Required:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the GitHub App.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the GitHub App.",
				Computed:            true,
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "The GitHub organization name.",
				Computed:            true,
			},
			"app_id": schema.Int64Attribute{
				MarkdownDescription: "The GitHub App ID.",
				Computed:            true,
			},
			"installation_id": schema.Int64Attribute{
				MarkdownDescription: "The GitHub App installation ID.",
				Computed:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The GitHub App client ID.",
				Computed:            true,
			},
			"webhook_secret": schema.StringAttribute{
				MarkdownDescription: "The GitHub App webhook secret, when returned by the Coolify API. Coolify may omit this value on read.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (d *gitHubAppDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *gitHubAppDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config gitHubAppDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_github_app"})

	app, err := d.client.GetGitHubApp(ctx, config.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitHub App", fmt.Sprintf("Could not read GitHub App %d: %s", config.ID.ValueInt64(), err))
		return
	}

	config.ID = types.Int64Value(app.ID)
	config.UUID = flex.StringToFramework(app.UUID)
	config.Name = types.StringValue(app.Name)
	config.OrganizationName = flex.StringToFramework(app.OrganizationName)
	config.AppID = types.Int64Value(app.AppID)
	config.InstallationID = types.Int64Value(app.InstallationID)
	config.ClientID = types.StringValue(app.ClientID)
	config.WebhookSecret = flex.StringToFramework(app.WebhookSecret)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

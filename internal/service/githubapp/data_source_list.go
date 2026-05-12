package githubapp

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*gitHubAppListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*gitHubAppListDataSource)(nil)
)

// gitHubAppListDataSource is the data source implementation for listing all Coolify GitHub Apps.
type gitHubAppListDataSource struct {
	client *client.Client
}

// gitHubAppListDataSourceModel maps the data source schema data.
type gitHubAppListDataSourceModel struct {
	GitHubApps []gitHubAppItemModel `tfsdk:"github_apps"`
}

// gitHubAppItemModel maps a single GitHub App in the list.
type gitHubAppItemModel struct {
	ID               types.Int64  `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	OrganizationName types.String `tfsdk:"organization_name"`
	AppID            types.Int64  `tfsdk:"app_id"`
	InstallationID   types.Int64  `tfsdk:"installation_id"`
	ClientID         types.String `tfsdk:"client_id"`
	WebhookSecret    types.String `tfsdk:"webhook_secret"`
}

// NewListDataSource returns a new GitHub Apps list data source instance.
func NewListDataSource() datasource.DataSource {
	return &gitHubAppListDataSource{}
}

func (d *gitHubAppListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_apps"
}

func (d *gitHubAppListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify GitHub App integrations.",
		Attributes: map[string]schema.Attribute{
			"github_apps": schema.ListNestedAttribute{
				MarkdownDescription: "The list of GitHub Apps.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The numeric identifier of the GitHub App.",
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
							MarkdownDescription: "The GitHub App webhook secret.",
							Computed:            true,
							Sensitive:           true,
						},
					},
				},
			},
		},
	}
}

func (d *gitHubAppListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data",
			"Expected *client.Client, got an unexpected type. Please report this issue to the provider developers.",
		)
		return
	}
	d.client = c
}

func (d *gitHubAppListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	apps, err := d.client.ListGitHubApps(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing GitHub Apps", fmt.Sprintf("Could not list GitHub Apps: %s", err))
		return
	}

	var state gitHubAppListDataSourceModel
	for _, a := range apps {
		item := gitHubAppItemModel{
			ID:               types.Int64Value(a.ID),
			Name:             types.StringValue(a.Name),
			OrganizationName: flex.StringToFramework(a.OrganizationName),
			AppID:            types.Int64Value(a.AppID),
			InstallationID:   types.Int64Value(a.InstallationID),
			ClientID:         types.StringValue(a.ClientID),
			WebhookSecret:    flex.StringToFramework(a.WebhookSecret),
		}
		state.GitHubApps = append(state.GitHubApps, item)
	}

	if state.GitHubApps == nil {
		state.GitHubApps = []gitHubAppItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

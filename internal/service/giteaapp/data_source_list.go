package giteaapp

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*listDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*listDataSource)(nil)
)

type listDataSource struct {
	client *client.Client
}

type listModel struct {
	Apps    []appModel      `tfsdk:"gitea_apps"`
	Filters []filter.Config `tfsdk:"filter"`
}

type appModel struct {
	ID           types.Int64  `tfsdk:"id"`
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	Organization types.String `tfsdk:"organization"`
	APIURL       types.String `tfsdk:"api_url"`
	HTMLURL      types.String `tfsdk:"html_url"`
	CustomUser   types.String `tfsdk:"custom_user"`
	CustomPort   types.Int64  `tfsdk:"custom_port"`
	ClientID     types.String `tfsdk:"client_id"`
	RedirectURI  types.String `tfsdk:"redirect_uri"`
	IsSystemWide types.Bool   `tfsdk:"is_system_wide"`
	IsPublic     types.Bool   `tfsdk:"is_public"`
	IsConnected  types.Bool   `tfsdk:"is_connected"`
	TeamID       types.Int64  `tfsdk:"team_id"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

// NewListDataSource returns the Gitea app list.
func NewListDataSource() datasource.DataSource {
	return &listDataSource{}
}

func (d *listDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gitea_apps"
}

func (d *listDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Gitea sources for the token's team, including system-wide sources. " +
			"Requires Coolify >= v4.7.0. `is_connected` is true when the OAuth connection can deploy private repositories. " +
			"Client secrets, access tokens, refresh tokens, and webhook secrets are not read into state. " +
			"Filter names: uuid, name, organization, api_url, html_url, client_id, is_connected, is_system_wide, is_public, and team_id.",
		Attributes: map[string]schema.Attribute{
			"gitea_apps": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Gitea sources visible to the token's team.",
				NestedObject:        schema.NestedAttributeObject{Attributes: appAttributes()},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func appAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":             schema.Int64Attribute{Computed: true, MarkdownDescription: "Numeric identifier."},
		"uuid":           schema.StringAttribute{Computed: true, MarkdownDescription: "UUID. Pass this as gitea_app_uuid when creating an application from the source."},
		"name":           schema.StringAttribute{Computed: true, MarkdownDescription: "Source name."},
		"organization":   schema.StringAttribute{Computed: true, MarkdownDescription: "Repository owners this source allows, comma-separated. Null when every owner is allowed."},
		"api_url":        schema.StringAttribute{Computed: true, MarkdownDescription: "Gitea API URL."},
		"html_url":       schema.StringAttribute{Computed: true, MarkdownDescription: "Gitea site URL."},
		"custom_user":    schema.StringAttribute{Computed: true, MarkdownDescription: "SSH user Coolify uses for this source. Null when Coolify stored none."},
		"custom_port":    schema.Int64Attribute{Computed: true, MarkdownDescription: "SSH port. Null when Coolify stored none."},
		"client_id":      schema.StringAttribute{Computed: true, MarkdownDescription: "OAuth client id. Null when the source has none."},
		"redirect_uri":   schema.StringAttribute{Computed: true, MarkdownDescription: "OAuth redirect URI. Null when the source has none."},
		"is_system_wide": schema.BoolAttribute{Computed: true, MarkdownDescription: "True when every team on the instance can use the source."},
		"is_public":      schema.BoolAttribute{Computed: true, MarkdownDescription: "True when the source is a public Gitea."},
		"is_connected":   schema.BoolAttribute{Computed: true, MarkdownDescription: "True when both the access token and the refresh token are stored."},
		"team_id":        schema.Int64Attribute{Computed: true, MarkdownDescription: "Owning team id. Null when Coolify did not return one."},
		"created_at":     schema.StringAttribute{Computed: true, MarkdownDescription: "When the source was created."},
		"updated_at":     schema.StringAttribute{Computed: true, MarkdownDescription: "When the source was last updated."},
	}
}

func (d *listDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *listDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data listModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !giteaCoolify47(d.client, &resp.Diagnostics) {
		return
	}
	apps, err := d.client.ListGiteaApps(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing Gitea apps", err.Error())
		return
	}
	apps = filter.Apply(ctx, apps, data.Filters, giteaFilterValue)
	data.Apps = make([]appModel, 0, len(apps))
	for i := range apps {
		data.Apps = append(data.Apps, flattenApp(apps[i]))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func giteaCoolify47(c *client.Client, diags *diag.Diagnostics) bool {
	if c != nil && !c.SupportsCoolify47() {
		diags.AddError("Coolify version", "Listing Gitea apps requires Coolify >= v4.7.0.")
		return false
	}
	return true
}

func flattenApp(app client.GiteaApp) appModel {
	return appModel{
		ID:           types.Int64Value(app.ID),
		UUID:         types.StringValue(app.UUID),
		Name:         types.StringValue(app.Name),
		Organization: strPtr(app.Organization),
		APIURL:       flex.StringToFramework(app.APIURL),
		HTMLURL:      flex.StringToFramework(app.HTMLURL),
		CustomUser:   flex.StringToFramework(app.CustomUser),
		CustomPort:   intPtr(app.CustomPort),
		ClientID:     strPtr(app.ClientID),
		RedirectURI:  strPtr(app.RedirectURI),
		IsSystemWide: types.BoolValue(app.IsSystemWide),
		IsPublic:     types.BoolValue(app.IsPublic),
		IsConnected:  types.BoolValue(app.IsConnected),
		TeamID:       intPtr(app.TeamID),
		CreatedAt:    flex.StringToFramework(app.CreatedAt),
		UpdatedAt:    flex.StringToFramework(app.UpdatedAt),
	}
}

func strPtr(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return flex.StringToFramework(*v)
}

func intPtr(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

func giteaFilterValue(app client.GiteaApp, field string) (string, bool) {
	switch field {
	case "uuid":
		return app.UUID, true
	case "name":
		return app.Name, true
	case "organization":
		return deref(app.Organization), true
	case "api_url":
		return app.APIURL, true
	case "html_url":
		return app.HTMLURL, true
	case "client_id":
		return deref(app.ClientID), true
	case "is_connected":
		return filter.BoolToString(app.IsConnected), true
	case "is_system_wide":
		return filter.BoolToString(app.IsSystemWide), true
	case "is_public":
		return filter.BoolToString(app.IsPublic), true
	case "team_id":
		if app.TeamID == nil {
			return "", true
		}
		return filter.Int64ToString(*app.TeamID), true
	default:
		return "", false
	}
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

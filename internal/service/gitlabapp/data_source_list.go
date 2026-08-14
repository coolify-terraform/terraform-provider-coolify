package gitlabapp

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = (*gitlabAppListDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*gitlabAppListDataSource)(nil)

type gitlabAppListDataSource struct{ client *client.Client }

type listModel struct {
	Apps []gitlabAppModel `tfsdk:"apps"`
}

func NewListDataSource() datasource.DataSource { return &gitlabAppListDataSource{} }

func (d *gitlabAppListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gitlab_apps"
}

func (d *gitlabAppListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Coolify GitLab Apps. Requires Coolify >= v4.3.0.",
		Attributes: map[string]schema.Attribute{
			"apps": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{Computed: true}, "uuid": schema.StringAttribute{Computed: true},
					"name": schema.StringAttribute{Computed: true}, "html_url": schema.StringAttribute{Computed: true},
					"api_url": schema.StringAttribute{Computed: true}, "custom_user": schema.StringAttribute{Computed: true},
					"custom_port": schema.Int64Attribute{Computed: true}, "group_name": schema.StringAttribute{Computed: true},
					"client_id": schema.StringAttribute{Computed: true}, "client_secret": schema.StringAttribute{Computed: true, Sensitive: true},
					"webhook_token": schema.StringAttribute{Computed: true, Sensitive: true}, "redirect_uri": schema.StringAttribute{Computed: true},
					"is_system_wide": schema.BoolAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *gitlabAppListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *gitlabAppListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data listModel
	apps, err := d.client.ListGitLabApps(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing GitLab Apps", err.Error())
		return
	}
	data.Apps = make([]gitlabAppModel, 0, len(apps))
	for i := range apps {
		var m gitlabAppModel
		flattenGitLab(&apps[i], &m)
		data.Apps = append(data.Apps, m)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

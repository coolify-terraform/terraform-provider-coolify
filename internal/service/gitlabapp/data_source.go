package gitlabapp

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = (*gitlabAppDS)(nil)
var _ datasource.DataSourceWithConfigure = (*gitlabAppDS)(nil)

type gitlabAppDS struct{ client *client.Client }

func NewDataSource() datasource.DataSource { return &gitlabAppDS{} }

func (d *gitlabAppDS) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gitlab_app"
}

func (d *gitlabAppDS) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Coolify GitLab App by id or uuid. Requires Coolify >= v4.3.0.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.Int64Attribute{Optional: true, Computed: true, MarkdownDescription: "Numeric Coolify id."},
			"uuid":           schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "UUID."},
			"name":           schema.StringAttribute{Computed: true},
			"html_url":       schema.StringAttribute{Computed: true},
			"api_url":        schema.StringAttribute{Computed: true},
			"custom_user":    schema.StringAttribute{Computed: true},
			"custom_port":    schema.Int64Attribute{Computed: true},
			"group_name":     schema.StringAttribute{Computed: true},
			"client_id":      schema.StringAttribute{Computed: true},
			"client_secret":  schema.StringAttribute{Computed: true, Sensitive: true},
			"webhook_token":  schema.StringAttribute{Computed: true, Sensitive: true},
			"redirect_uri":   schema.StringAttribute{Computed: true},
			"is_system_wide": schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *gitlabAppDS) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *gitlabAppDS) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gitlabAppModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var got *client.GitLabApp
	var err error
	switch {
	case !data.ID.IsNull() && !data.ID.IsUnknown() && data.ID.ValueInt64() != 0:
		got, err = d.client.GetGitLabApp(ctx, data.ID.ValueInt64())
	case !data.UUID.IsNull() && data.UUID.ValueString() != "":
		got, err = d.client.GetGitLabAppByUUID(ctx, data.UUID.ValueString())
	default:
		resp.Diagnostics.AddError("Invalid configuration", "set id or uuid")
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading GitLab App", err.Error())
		return
	}
	flattenGitLab(got, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

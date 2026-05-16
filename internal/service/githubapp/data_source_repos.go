package githubapp

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*gitHubAppReposDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*gitHubAppReposDataSource)(nil)
)

// gitHubAppReposDataSource is the data source implementation for listing repositories
// accessible to a Coolify GitHub App.
type gitHubAppReposDataSource struct {
	client *client.Client
}

// gitHubAppReposDataSourceModel maps the data source schema data.
type gitHubAppReposDataSourceModel struct {
	GitHubAppID  types.Int64           `tfsdk:"github_app_id"`
	Repositories []gitHubRepoItemModel `tfsdk:"repositories"`
	Filters      []filter.Config       `tfsdk:"filter"`
}

// gitHubRepoItemModel maps a single repository in the list.
type gitHubRepoItemModel struct {
	Name     types.String `tfsdk:"name"`
	FullName types.String `tfsdk:"full_name"`
	Private  types.Bool   `tfsdk:"private"`
}

// NewReposDataSource returns a new GitHub App repositories data source instance.
func NewReposDataSource() datasource.DataSource {
	return &gitHubAppReposDataSource{}
}

func (d *gitHubAppReposDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_app_repositories"
}

func (d *gitHubAppReposDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list repositories accessible to a Coolify GitHub App.",
		Attributes: map[string]schema.Attribute{
			"github_app_id": schema.Int64Attribute{
				MarkdownDescription: "The numeric identifier of the GitHub App.",
				Required:            true,
			},
			"repositories": schema.ListNestedAttribute{
				MarkdownDescription: "The list of repositories.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The repository name.",
							Computed:            true,
						},
						"full_name": schema.StringAttribute{
							MarkdownDescription: "The full repository name (owner/repo).",
							Computed:            true,
						},
						"private": schema.BoolAttribute{
							MarkdownDescription: "Whether the repository is private.",
							Computed:            true,
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *gitHubAppReposDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *gitHubAppReposDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config gitHubAppReposDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repos, err := d.client.ListGitHubAppRepositories(ctx, config.GitHubAppID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error listing repositories", fmt.Sprintf("Could not list repositories: %s", err))
		return
	}

	repos = filter.Apply(repos, config.Filters, func(r client.GitHubRepository, field string) (string, bool) {
		switch field {
		case "name":
			return r.Name, true
		case "full_name":
			return r.FullName, true
		case "private":
			return filter.BoolToString(r.Private), true
		default:
			return "", false
		}
	})

	for _, r := range repos {
		config.Repositories = append(config.Repositories, gitHubRepoItemModel{
			Name:     types.StringValue(r.Name),
			FullName: types.StringValue(r.FullName),
			Private:  types.BoolValue(r.Private),
		})
	}

	if config.Repositories == nil {
		config.Repositories = []gitHubRepoItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

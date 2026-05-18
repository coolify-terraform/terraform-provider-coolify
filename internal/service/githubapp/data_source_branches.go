package githubapp

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*gitHubAppBranchesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*gitHubAppBranchesDataSource)(nil)
)

// gitHubAppBranchesDataSource is the data source implementation for listing branches
// of a repository accessible to a Coolify GitHub App.
type gitHubAppBranchesDataSource struct {
	client *client.Client
}

// gitHubAppBranchesDataSourceModel maps the data source schema data.
type gitHubAppBranchesDataSourceModel struct {
	GitHubAppID types.Int64             `tfsdk:"github_app_id"`
	Owner       types.String            `tfsdk:"owner"`
	Repo        types.String            `tfsdk:"repo"`
	Branches    []gitHubBranchItemModel `tfsdk:"branches"`
	Filters     []filter.Config         `tfsdk:"filter"`
}

// gitHubBranchItemModel maps a single branch in the list.
type gitHubBranchItemModel struct {
	Name types.String `tfsdk:"name"`
}

// NewBranchesDataSource returns a new GitHub App branches data source instance.
func NewBranchesDataSource() datasource.DataSource {
	return &gitHubAppBranchesDataSource{}
}

func (d *gitHubAppBranchesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_app_branches"
}

func (d *gitHubAppBranchesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list branches for a repository accessible to a Coolify GitHub App.",
		Attributes: map[string]schema.Attribute{
			"github_app_id": schema.Int64Attribute{
				MarkdownDescription: "The numeric identifier of the GitHub App.",
				Required:            true,
			},
			"owner": schema.StringAttribute{
				MarkdownDescription: "The repository owner.",
				Required:            true,
			},
			"repo": schema.StringAttribute{
				MarkdownDescription: "The repository name.",
				Required:            true,
			},
			"branches": schema.ListNestedAttribute{
				MarkdownDescription: "The list of branches.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The branch name.",
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

func (d *gitHubAppBranchesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *gitHubAppBranchesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config gitHubAppBranchesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_github_app_branches"})

	branches, err := d.client.ListGitHubAppBranches(
		ctx,
		config.GitHubAppID.ValueInt64(),
		config.Owner.ValueString(),
		config.Repo.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error listing branches", fmt.Sprintf("Could not list branches: %s", err))
		return
	}

	branches = filter.Apply(ctx, branches, config.Filters, func(b client.GitHubBranch, field string) (string, bool) {
		switch field {
		case "name":
			return b.Name, true
		default:
			return "", false
		}
	})

	for _, b := range branches {
		config.Branches = append(config.Branches, gitHubBranchItemModel{
			Name: types.StringValue(b.Name),
		})
	}

	if config.Branches == nil {
		config.Branches = []gitHubBranchItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

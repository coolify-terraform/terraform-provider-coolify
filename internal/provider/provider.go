//go:generate tfplugindocs generate --provider-name coolify --provider-dir ../.. --rendered-provider-name coolify
package provider

import (
	"context"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/application"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/mariadb"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/mongodb"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/mysql"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/postgresql"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/redis"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/deployment"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/environmentvariable"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/privatekey"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/project"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/server"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/service"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/team"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"os"
)

var _ provider.Provider = (*coolifyProvider)(nil)

type coolifyProvider struct{ version string }
type coolifyProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &coolifyProvider{version: version} }
}
func (p *coolifyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "coolify"
	resp.Version = p.version
}
func (p *coolifyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "The Coolify provider manages resources in a Coolify instance.", Attributes: map[string]schema.Attribute{
		"endpoint": schema.StringAttribute{MarkdownDescription: "Coolify API endpoint. Env: COOLIFY_ENDPOINT.", Optional: true},
		"token":    schema.StringAttribute{MarkdownDescription: "Coolify API token. Env: COOLIFY_TOKEN.", Optional: true, Sensitive: true},
	}}
}
func (p *coolifyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config coolifyProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	endpoint := os.Getenv("COOLIFY_ENDPOINT")
	if !config.Endpoint.IsNull() && !config.Endpoint.IsUnknown() {
		endpoint = config.Endpoint.ValueString()
	}
	if endpoint == "" {
		resp.Diagnostics.AddError("Missing Coolify Endpoint", "Set endpoint in provider block or COOLIFY_ENDPOINT env var.")
	}
	token := os.Getenv("COOLIFY_TOKEN")
	if !config.Token.IsNull() && !config.Token.IsUnknown() {
		token = config.Token.ValueString()
	}
	if token == "" {
		resp.Diagnostics.AddError("Missing Coolify Token", "Set token in provider block or COOLIFY_TOKEN env var.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	c := client.New(endpoint, token)
	resp.DataSourceData = c
	resp.ResourceData = c
}
func (p *coolifyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{application.NewResource, application.NewDockerResource, application.NewPrivateGitResource, deployment.NewResource, environmentvariable.NewResource, postgresql.NewResource, mysql.NewResource, mariadb.NewResource, redis.NewResource, mongodb.NewResource, privatekey.NewResource, project.NewResource, server.NewResource, service.NewResource}
}
func (p *coolifyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{application.NewDataSource, application.NewListDataSource, database.NewListDataSource, project.NewDataSource, project.NewListDataSource, server.NewDataSource, server.NewListDataSource, service.NewListDataSource, privatekey.NewDataSource, privatekey.NewListDataSource, team.NewDataSource}
}

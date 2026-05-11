//go:generate tfplugindocs generate --provider-name coolify --provider-dir ../.. --rendered-provider-name coolify
package provider

import (
	"context"
	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/application"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/cloudtoken"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/backup"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/clickhouse"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/dragonfly"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/keydb"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/mariadb"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/mongodb"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/mysql"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/postgresql"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/redis"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/deployment"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/environment"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/environmentvariable"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/githubapp"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/health"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/hetzner"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/privatekey"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/project"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/resourcelist"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/s3storage"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/scheduledtask"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/server"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/service"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/storage"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/team"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/version"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"os"
	"strings"
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
	switch {
	case endpoint == "":
		resp.Diagnostics.AddError("Missing Coolify Endpoint", "Set endpoint in provider block or COOLIFY_ENDPOINT env var.")
	case !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://"):
		resp.Diagnostics.AddError("Invalid Coolify Endpoint", "Endpoint must start with http:// or https://.")
	case strings.HasPrefix(endpoint, "http://"):
		host := strings.TrimPrefix(endpoint, "http://")
		host = strings.SplitN(host, "/", 2)[0]
		host = strings.SplitN(host, ":", 2)[0]
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			resp.Diagnostics.AddWarning(
				"Insecure Coolify Endpoint",
				"The endpoint uses plain HTTP. The API token will be sent in cleartext. Use https:// for non-local endpoints.",
			)
		}
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
	endpoint = strings.TrimRight(endpoint, "/")
	c := client.New(endpoint, token)
	if p.version != "" {
		c.UserAgent = "terraform-provider-coolify/" + p.version
	}

	// Validate the connection by fetching the Coolify version.
	if _, err := c.GetVersion(ctx); err != nil {
		resp.Diagnostics.AddError(
			"Unable to connect to Coolify",
			"The provider could not reach the Coolify API at "+endpoint+". "+
				"Verify that the endpoint is correct, the server is running, "+
				"and the API token is valid.\n\nError: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}
func (p *coolifyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{application.NewResource, application.NewDockerResource, application.NewDockerComposeResource, application.NewPrivateGitResource, application.NewDockerfileResource, application.NewGitHubAppResource, backup.NewResource, cloudtoken.NewResource, deployment.NewResource, environment.NewResource, environmentvariable.NewResource, githubapp.NewResource, postgresql.NewResource, mysql.NewResource, mariadb.NewResource, redis.NewResource, mongodb.NewResource, clickhouse.NewResource, keydb.NewResource, dragonfly.NewResource, privatekey.NewResource, project.NewResource, s3storage.NewResource, scheduledtask.NewResource, server.NewResource, service.NewResource, storage.NewResource}
}
func (p *coolifyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{application.NewDataSource, application.NewListDataSource, application.NewLogsDataSource, backup.NewExecutionsDataSource, cloudtoken.NewDataSource, cloudtoken.NewListDataSource, database.NewListDataSource, database.NewDataSource, deployment.NewListDataSource, environment.NewDataSource, environment.NewListDataSource, environmentvariable.NewListDataSource, githubapp.NewListDataSource, githubapp.NewReposDataSource, githubapp.NewBranchesDataSource, health.NewDataSource, hetzner.NewImagesDataSource, hetzner.NewLocationsDataSource, hetzner.NewServerTypesDataSource, hetzner.NewSSHKeysDataSource, project.NewDataSource, project.NewListDataSource, resourcelist.NewDataSource, s3storage.NewDataSource, s3storage.NewListDataSource, scheduledtask.NewListDataSource, scheduledtask.NewExecutionsDataSource, server.NewDataSource, server.NewListDataSource, server.NewResourcesDataSource, server.NewDomainsDataSource, server.NewValidateDataSource, service.NewListDataSource, service.NewDataSource, privatekey.NewDataSource, privatekey.NewListDataSource, storage.NewListDataSource, team.NewDataSource, team.NewListDataSource, team.NewMembersDataSource, version.NewDataSource}
}

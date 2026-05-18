//go:generate tfplugindocs generate --provider-name coolify --provider-dir ../.. --rendered-provider-name coolify
package provider

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/application"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/cloudtoken"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/backup"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/clickhouse"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/dragonfly"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/keydb"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/mariadb"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/mongodb"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/mysql"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/postgresql"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/redis"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/deployment"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/environment"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/environmentvariable"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/githubapp"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/health"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/hetzner"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/privatekey"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/project"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/resourcelist"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/scheduledtask"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/server"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/service"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/storage"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/team"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/service/version"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const minCoolifyVersion = "4.0.0"

var _ provider.Provider = (*coolifyProvider)(nil)

type coolifyProvider struct{ version string }
type coolifyProviderModel struct {
	Endpoint     types.String `tfsdk:"endpoint"`
	Token        types.String `tfsdk:"token"`
	RetryMax     types.Int64  `tfsdk:"retry_max"`
	RetryMinWait types.Int64  `tfsdk:"retry_min_wait"`
	RetryMaxWait types.Int64  `tfsdk:"retry_max_wait"`
	CACert       types.String `tfsdk:"ca_cert"`
	Insecure     types.Bool   `tfsdk:"insecure"`
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
		"endpoint":       schema.StringAttribute{MarkdownDescription: "Coolify API endpoint. Env: COOLIFY_ENDPOINT.", Optional: true},
		"token":          schema.StringAttribute{MarkdownDescription: "Coolify API token. Env: COOLIFY_TOKEN.", Optional: true, Sensitive: true},
		"retry_max":      schema.Int64Attribute{MarkdownDescription: "Maximum number of API request retries (default: 3).", Optional: true},
		"retry_min_wait": schema.Int64Attribute{MarkdownDescription: "Minimum wait between retries in seconds (default: 1).", Optional: true},
		"retry_max_wait": schema.Int64Attribute{MarkdownDescription: "Maximum wait between retries in seconds (default: 30).", Optional: true},
		"ca_cert":        schema.StringAttribute{MarkdownDescription: "PEM-encoded CA certificate to trust for TLS connections to the Coolify API. Use this when your Coolify instance uses a self-signed certificate or an internal CA. Env: `COOLIFY_CA_CERT`.", Optional: true},
		"insecure":       schema.BoolAttribute{MarkdownDescription: "Skip TLS certificate verification. **Not recommended for production.** Use `ca_cert` instead when possible. Env: `COOLIFY_INSECURE`.", Optional: true},
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
	cfg := buildClientConfig(config)
	if cfg.Insecure && cfg.CACert != "" {
		resp.Diagnostics.AddWarning("CA certificate ignored",
			"Both insecure and ca_cert are set. When insecure is true, "+
				"TLS certificate verification is skipped entirely and ca_cert is not used.")
	}
	if cfg.CACert != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(cfg.CACert)) {
			resp.Diagnostics.AddError("Invalid CA Certificate",
				"The ca_cert value could not be parsed as a PEM-encoded certificate. "+
					"Check that the value is a valid PEM block starting with -----BEGIN CERTIFICATE-----.")
			return
		}
	}
	c := client.New(endpoint, token, cfg)
	if p.version != "" {
		c.UserAgent = "terraform-provider-coolify/" + p.version
	}

	// Validate the connection and Coolify version.
	coolifyVersion, err := c.GetVersion(ctx)
	if err != nil {
		diagnosticEndpoint := redactEndpointForDiagnostics(endpoint)
		resp.Diagnostics.AddError(
			"Unable to connect to Coolify",
			"The provider could not reach the Coolify API at "+diagnosticEndpoint+". "+
				"Verify that the endpoint is correct, the server is running, "+
				"and the API token is valid.\n\nError: "+err.Error(),
		)
		return
	}
	if !isVersionAtLeast(coolifyVersion, minCoolifyVersion) {
		resp.Diagnostics.AddError(
			"Unsupported Coolify version",
			fmt.Sprintf("The connected Coolify instance is running %s, but this provider requires %s or later. Please upgrade your Coolify instance.", coolifyVersion, minCoolifyVersion),
		)
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func redactEndpointForDiagnostics(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.User == nil {
		return endpoint
	}
	if _, hasPassword := parsed.User.Password(); hasPassword {
		parsed.User = url.UserPassword("REDACTED", "REDACTED")
	} else {
		parsed.User = url.User("REDACTED")
	}
	return parsed.String()
}

func (p *coolifyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Applications (sorted by type suffix).
		application.NewResource,
		application.NewDockerResource,
		application.NewDockerfileResource,
		application.NewGitHubAppResource,
		application.NewPrivateGitResource,
		// Databases (sorted alphabetically).
		clickhouse.NewResource,
		dragonfly.NewResource,
		keydb.NewResource,
		mariadb.NewResource,
		mongodb.NewResource,
		mysql.NewResource,
		postgresql.NewResource,
		redis.NewResource,
		// Other resources (sorted alphabetically by type name).
		backup.NewResource,
		cloudtoken.NewResource,
		deployment.NewResource,
		environment.NewResource,
		environmentvariable.NewResource,
		githubapp.NewResource,
		hetzner.NewResource,
		privatekey.NewResource,
		project.NewResource,
		scheduledtask.NewResource,
		server.NewResource,
		service.NewResource,
		storage.NewResource,
	}
}

func (p *coolifyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		application.NewDataSource,
		application.NewListDataSource,
		application.NewLogsDataSource,
		backup.NewExecutionsDataSource,
		cloudtoken.NewDataSource,
		cloudtoken.NewListDataSource,
		database.NewDataSource,
		database.NewListDataSource,
		deployment.NewDataSource,
		deployment.NewListDataSource,
		environment.NewDataSource,
		environment.NewListDataSource,
		environmentvariable.NewDataSource,
		environmentvariable.NewListDataSource,
		githubapp.NewBranchesDataSource,
		githubapp.NewDataSource,
		githubapp.NewListDataSource,
		githubapp.NewReposDataSource,
		health.NewDataSource,
		hetzner.NewImagesDataSource,
		hetzner.NewLocationsDataSource,
		hetzner.NewServerTypesDataSource,
		hetzner.NewSSHKeysDataSource,
		privatekey.NewDataSource,
		privatekey.NewListDataSource,
		project.NewDataSource,
		project.NewListDataSource,
		resourcelist.NewDataSource,
		scheduledtask.NewDataSource,
		scheduledtask.NewExecutionsDataSource,
		scheduledtask.NewListDataSource,
		server.NewDataSource,
		server.NewDomainsDataSource,
		server.NewListDataSource,
		server.NewResourcesDataSource,
		server.NewValidateDataSource,
		service.NewDataSource,
		service.NewListDataSource,
		storage.NewDataSource,
		storage.NewListDataSource,
		team.NewDataSource,
		team.NewListDataSource,
		team.NewMembersDataSource,
		version.NewDataSource,
	}
}

func buildClientConfig(config coolifyProviderModel) client.RetryConfig {
	var cfg client.RetryConfig
	if !config.RetryMax.IsNull() {
		cfg.Attempts = int(config.RetryMax.ValueInt64())
	}
	if !config.RetryMinWait.IsNull() {
		cfg.MinWait = time.Duration(config.RetryMinWait.ValueInt64()) * time.Second
	}
	if !config.RetryMaxWait.IsNull() {
		cfg.MaxWait = time.Duration(config.RetryMaxWait.ValueInt64()) * time.Second
	}
	cfg.CACert = os.Getenv("COOLIFY_CA_CERT")
	if !config.CACert.IsNull() && !config.CACert.IsUnknown() {
		cfg.CACert = config.CACert.ValueString()
	}
	if !config.Insecure.IsNull() && !config.Insecure.IsUnknown() {
		cfg.Insecure = config.Insecure.ValueBool()
	} else if strings.EqualFold(os.Getenv("COOLIFY_INSECURE"), "true") {
		cfg.Insecure = true
	}
	return cfg
}

// isVersionAtLeast compares two semver-like version strings (e.g. "4.0.0").
// Returns true if actual >= minimum. Non-parseable versions return true
// to avoid blocking on unexpected version formats.
func isVersionAtLeast(actual, minimum string) bool {
	parse := func(v string) (int, int, int, bool) {
		v = strings.TrimPrefix(v, "v")
		parts := strings.SplitN(v, ".", 3)
		if len(parts) < 2 {
			return 0, 0, 0, false
		}
		major, err1 := strconv.Atoi(parts[0])
		minor, err2 := strconv.Atoi(parts[1])
		patch := 0
		if len(parts) == 3 {
			// Strip pre-release suffix (e.g. "0-beta.335")
			p := strings.SplitN(parts[2], "-", 2)[0]
			patch, _ = strconv.Atoi(p)
		}
		if err1 != nil || err2 != nil {
			return 0, 0, 0, false
		}
		return major, minor, patch, true
	}
	aMaj, aMin, aPat, aOk := parse(actual)
	mMaj, mMin, mPat, mOk := parse(minimum)
	if !aOk || !mOk {
		return true
	}
	if aMaj != mMaj {
		return aMaj > mMaj
	}
	if aMin != mMin {
		return aMin > mMin
	}
	return aPat >= mPat
}

package application

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &ApplicationDataSource{}
	_ datasource.DataSourceWithConfigure = &ApplicationDataSource{}
)

// ApplicationDataSource reads a single Coolify application.
type ApplicationDataSource struct {
	client *client.Client
}

// ApplicationDataSourceModel maps the data source schema to Go types.
type ApplicationDataSourceModel struct {
	UUID                             types.String `tfsdk:"uuid"`
	Name                             types.String `tfsdk:"name"`
	Description                      types.String `tfsdk:"description"`
	Domains                          types.String `tfsdk:"domains"`
	GitRepository                    types.String `tfsdk:"git_repository"`
	GitBranch                        types.String `tfsdk:"git_branch"`
	BuildPack                        types.String `tfsdk:"build_pack"`
	DockerfileLocation               types.String `tfsdk:"dockerfile_location"`
	InstallCommand                   types.String `tfsdk:"install_command"`
	BuildCommand                     types.String `tfsdk:"build_command"`
	StartCommand                     types.String `tfsdk:"start_command"`
	PortsExposes                     types.String `tfsdk:"ports_exposes"`
	ProjectUUID                      types.String `tfsdk:"project_uuid"`
	ServerUUID                       types.String `tfsdk:"server_uuid"`
	EnvironmentName                  types.String `tfsdk:"environment_name"`
	Status                           types.String `tfsdk:"status"`
	DockerComposeRaw                 types.String `tfsdk:"docker_compose_raw"`
	DockerRegistryImageName          types.String `tfsdk:"docker_registry_image_name"`
	MaxRestartCount                  types.Int64  `tfsdk:"max_restart_count"`
	RestartLimitReached              types.Bool   `tfsdk:"restart_limit_reached"`
	ContainerPresent                 types.Bool   `tfsdk:"container_present"`
	IsConsistentContainerNameEnabled types.Bool   `tfsdk:"is_consistent_container_name_enabled"`
	NoindexDomains                   types.List   `tfsdk:"noindex_domains"`
}

// NewDataSource returns a new ApplicationDataSource instance.
func NewDataSource() datasource.DataSource {
	return &ApplicationDataSource{}
}

func (d *ApplicationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (d *ApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to retrieve information about a Coolify application by UUID.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application to look up.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the application.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the application.",
				Computed:            true,
			},
			"domains": schema.StringAttribute{
				MarkdownDescription: "The fully qualified domain name of the application.",
				Computed:            true,
			},
			"git_repository": schema.StringAttribute{
				MarkdownDescription: "The Git repository URL of the application.",
				Computed:            true,
			},
			"git_branch": schema.StringAttribute{
				MarkdownDescription: "The Git branch used by the application.",
				Computed:            true,
			},
			"build_pack": schema.StringAttribute{
				MarkdownDescription: "The build pack type used by the application.",
				Computed:            true,
			},
			"dockerfile_location": schema.StringAttribute{
				MarkdownDescription: "For Git-backed applications, the path to the Dockerfile relative to the repository root. For `coolify_application_dockerfile` resources, this contains the Dockerfile content (base64-encoded).",
				Computed:            true,
			},
			"install_command": schema.StringAttribute{
				MarkdownDescription: "The install command.",
				Computed:            true,
			},
			"build_command": schema.StringAttribute{
				MarkdownDescription: "The build command.",
				Computed:            true,
			},
			"start_command": schema.StringAttribute{
				MarkdownDescription: "The start command.",
				Computed:            true,
			},
			"ports_exposes": schema.StringAttribute{
				MarkdownDescription: "The exposed ports.",
				Computed:            true,
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project the application belongs to.",
				Computed:            true,
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server the application is deployed on.",
				Computed:            true,
			},
			"environment_name": schema.StringAttribute{
				MarkdownDescription: "The environment name of the application.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the application (e.g., running, stopped, exited).",
				Computed:            true,
			},
			"docker_compose_raw": schema.StringAttribute{
				MarkdownDescription: "The raw Docker Compose content.",
				Computed:            true,
				Sensitive:           true,
			},
			"docker_registry_image_name": schema.StringAttribute{
				MarkdownDescription: "The Docker registry image name.",
				Computed:            true,
			},
			"max_restart_count": schema.Int64Attribute{
				MarkdownDescription: "The maximum number of container restarts before Coolify stops the application. Computed; may be null if the API omits it.",
				Computed:            true,
			},
			"restart_limit_reached": schema.BoolAttribute{
				MarkdownDescription: "Whether Coolify has stopped the application because restart_count reached max_restart_count. Coolify tip after 2026-08-31 (not in tag v4.3.14).",
				Computed:            true,
			},
			"container_present": schema.BoolAttribute{
				MarkdownDescription: "Whether Coolify last observed the application container on the server. Coolify tip after 2026-08-31 (not in tag v4.3.14).",
				Computed:            true,
			},
			"is_consistent_container_name_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether Coolify uses a consistent container name for this application. Coolify default is `false`. " +
					"Set to `true` for apps that keep an exclusive file lock on a persistent volume (SQLite, DuckDB, LMDB, BoltDB). " +
					"A fixed name makes Docker refuse a second container on the same mounts, so Coolify falls back to stop-then-start instead of a rolling update that would leave the new container unable to open the store while still reporting a successful deploy. " +
					"Requires Coolify >= v4.3.0.",
				Computed: true,
			},
			"noindex_domains": schema.ListAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "Subset of application domain URLs served with an `X-Robots-Tag: noindex, nofollow` response header " +
					"(keeps them out of search engines). Entries that are not among the application domains are ignored by Coolify. " +
					"Requires Coolify >= v4.3.0. Omitted or empty on GET from older Coolify instances.",
				Computed: true,
			},
		},
	}
}

func (d *ApplicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *ApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ApplicationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_application"})

	app, err := d.client.GetApplication(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading application", fmt.Sprintf("application %s: %s", config.UUID.ValueString(), err))
		return
	}

	config.UUID = types.StringValue(app.UUID)
	config.Name = flex.StringToFramework(app.Name)
	config.Description = flex.StringToFramework(app.Description)
	config.Domains = flex.StringToFramework(app.Domains)
	config.GitRepository = flex.StringToFramework(app.GitRepository)
	config.GitBranch = flex.StringToFramework(app.GitBranch)
	config.BuildPack = flex.StringToFramework(app.BuildPack)
	config.DockerfileLocation = flex.StringToFramework(app.DockerfileLocation)
	config.InstallCommand = flex.StringToFramework(app.InstallCommand)
	config.BuildCommand = flex.StringToFramework(app.BuildCommand)
	config.StartCommand = flex.StringToFramework(app.StartCommand)
	config.PortsExposes = flex.StringToFramework(app.PortsExposes)
	config.ProjectUUID = flex.StringToFramework(app.ProjectUUID)
	config.ServerUUID = flex.StringToFramework(app.ServerUUID)
	config.EnvironmentName = flex.StringToFramework(app.EnvironmentName)
	config.Status = flex.StringToFramework(app.Status)
	config.DockerComposeRaw = flex.StringToFramework(app.DockerComposeRaw)
	config.DockerRegistryImageName = flex.StringToFramework(app.DockerRegistryImageName)
	flattenRestartLimitFields(app, commonAppFields{
		MaxRestartCount:     &config.MaxRestartCount,
		RestartLimitReached: &config.RestartLimitReached,
		ContainerPresent:    &config.ContainerPresent,
	})
	if app.IsConsistentContainerNameEnabled != nil {
		config.IsConsistentContainerNameEnabled = types.BoolValue(*app.IsConsistentContainerNameEnabled)
	} else {
		config.IsConsistentContainerNameEnabled = types.BoolValue(false)
	}
	config.NoindexDomains = types.ListNull(types.StringType)
	flattenNoindexDomains(app.NoindexDomains, &config.NoindexDomains)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

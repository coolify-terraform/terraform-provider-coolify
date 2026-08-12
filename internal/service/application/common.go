package application

import (
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Application defaults — single source of truth for schema, import, and flatten.
const (
	defaultRedirect        = "both"
	defaultStaticImage     = "nginx:alpine"
	defaultHealthCheckHost = "localhost"
	defaultHealthCheckType = "http"
	defaultHealthCheckMeth = "GET"
	defaultHealthCheckSchm = "http"
	defaultHealthCheckCode = int64(200)
)

// coolifyDockerComposeDomainsNeedRaw is the Coolify API error substring when
// docker_compose_domains is set before docker_compose_raw exists. Present on
// all Coolify versions supported by this provider (v4.1.0 through v4.2.0+).
const coolifyDockerComposeDomainsNeedRaw = "Cannot set docker_compose_domains without docker_compose_raw"

// dockerComposeDomainsDescription is the shared schema docs for
// docker_compose_domains on application resources. Kept as a constant so unit
// tests can lock the ordering constraint and write shape for every supported
// Coolify version without regenerating provider docs in the test.
const dockerComposeDomainsDescription = "Domain mappings for Docker Compose services (`build_pack = \"dockercompose\"`). " +
	"Send a JSON array of objects with `name` (compose service name) and `domain` (comma-separated http(s) URLs), for example " +
	"`jsonencode([{ name = \"web\", domain = \"https://app.example.com\" }])`. " +
	"Coolify accepts only that array form on write, stores an object map keyed by service name, and returns the object form on GET; " +
	"the provider normalizes both shapes so Terraform plans stay empty after apply. " +
	"Coolify rejects this field until `docker_compose_raw` is set. For git-sourced compose apps, Coolify only populates " +
	"`docker_compose_raw` after a deployment loads the compose file from the repository; there is no separate load-compose API. " +
	"This ordering is a Coolify API constraint on all Coolify versions supported by this provider (v4.1.0 and later). " +
	"Recommended two-stage apply: (1) create without `docker_compose_domains` and deploy once (`instant_deploy = true` or a manual deploy), " +
	"wait until the deployment succeeds; (2) add `docker_compose_domains` and apply again. " +
	"Alternatively use `coolify_service` with inline `docker_compose_raw` when the compose file can live in Terraform."

// annotateDockerComposeDomainsError appends operator guidance when Coolify
// rejects docker_compose_domains because compose raw is not loaded yet.
func annotateDockerComposeDomainsError(err error) string {
	if err == nil {
		return ""
	}
	if !strings.Contains(err.Error(), coolifyDockerComposeDomainsNeedRaw) {
		return ""
	}
	return " Coolify requires docker_compose_raw before docker_compose_domains on all supported Coolify versions (v4.1.0+). " +
		"For git-sourced dockercompose applications, deploy once so Coolify loads the compose file, then set " +
		"docker_compose_domains on a second apply. See the docker_compose_domains attribute docs and the Common Errors guide."
}

// commonAppFields holds pointers to the fields shared by all application
// resource models. This allows a single flatten function to write into
// any concrete model type.
type commonAppFields struct {
	UUID               *types.String
	Name               *types.String
	Description        *types.String
	GitRepository      *types.String
	GitBranch          *types.String
	BuildPack          *types.String
	PortsExposes       *types.String
	Domains            *types.String
	DockerfileLocation *types.String
	InstallCommand     *types.String
	BuildCommand       *types.String
	StartCommand       *types.String
	Status             *types.String
	ProjectUUID        *types.String
	ServerUUID         *types.String
	EnvironmentName    *types.String
	// Resource limits
	LimitsMemory            *types.String
	LimitsMemorySwap        *types.String
	LimitsMemorySwappiness  *types.Int64
	LimitsMemoryReservation *types.String
	LimitsCPUs              *types.String
	LimitsCPUSet            *types.String
	LimitsCPUShares         *types.Int64
	// Health checks
	HealthCheckEnabled     *types.Bool
	HealthCheckPath        *types.String
	HealthCheckPort        *types.String
	HealthCheckInterval    *types.Int64
	HealthCheckTimeout     *types.Int64
	HealthCheckRetries     *types.Int64
	HealthCheckStartPeriod *types.Int64
	// Extended health checks
	HealthCheckCommand      *types.String
	HealthCheckHost         *types.String
	HealthCheckMethod       *types.String
	HealthCheckResponseText *types.String
	HealthCheckReturnCode   *types.Int64
	HealthCheckScheme       *types.String
	HealthCheckType         *types.String
	// Auto-deploy
	IsAutoDeployEnabled *types.Bool
	// Extended build/deploy settings
	BaseDirectory                   *types.String
	Dockerfile                      *types.String
	DockerfileTargetBuild           *types.String
	DockerRegistryImageTag          *types.String
	DockerComposeLocation           *types.String
	DockerComposeCustomBuildCommand *types.String
	DockerComposeCustomStartCommand *types.String
	DockerComposeDomains            *types.String
	GitCommitSha                    *types.String
	PublishDirectory                *types.String
	WatchPaths                      *types.String
	PreviewURLTemplate              *types.String
	// Container/Network settings
	CustomDockerRunOptions   *types.String
	CustomLabels             *types.String
	CustomNetworkAliases     *types.String
	CustomNginxConfiguration *types.String
	PortsMappings            *types.String
	ConnectToDockerNetwork   *types.Bool
	// Redirect & static
	Redirect    *types.String
	StaticImage *types.String
	IsStatic    *types.Bool
	IsSPA       *types.Bool
	// Security & Auth
	IsForceHTTPSEnabled    *types.Bool
	IsHTTPBasicAuthEnabled *types.Bool
	HTTPBasicAuthUsername  *types.String
	HTTPBasicAuthPassword  *types.String
	// Deployment commands
	PreDeploymentCommand           *types.String
	PreDeploymentCommandContainer  *types.String
	PostDeploymentCommand          *types.String
	PostDeploymentCommandContainer *types.String
	// Webhook secrets
	ManualWebhookSecretBitbucket *types.String
	ManualWebhookSecretGitea     *types.String
	ManualWebhookSecretGitHub    *types.String
	ManualWebhookSecretGitLab    *types.String
	// Other settings
	ForceDomainOverride           *types.Bool
	IsContainerLabelEscapeEnabled *types.Bool
	IsPreserveRepositoryEnabled   *types.Bool
	UseBuildServer                *types.Bool
	IsPreviewDeploymentsEnabled   *types.Bool
	UseBuildSecrets               *types.Bool
	StopGracePeriod               *types.Int64
	IsGitSubmodulesEnabled        *types.Bool
	IsGitLfsEnabled               *types.Bool
	IsGitShallowCloneEnabled      *types.Bool
	DisableBuildCache             *types.Bool
	InjectBuildArgsToDockerfile   *types.Bool
	IncludeSourceCommitInBuild    *types.Bool
	IsEnvSortingEnabled           *types.Bool
	IsPrDeploymentsPublicEnabled  *types.Bool
	DockerImagesToKeep            *types.Int64
	IsGzipEnabled                 *types.Bool
	IsStripprefixEnabled          *types.Bool
	IsRawComposeDeploymentEnabled *types.Bool
	// Coolify >= v4.3.0 APPLICATION_SETTING_FIELDS + noindex_domains.
	IsLogDrainEnabled                *types.Bool
	IsGpuEnabled                     *types.Bool
	GpuDriver                        *types.String
	GpuCount                         *types.String
	GpuDeviceIds                     *types.String
	GpuOptions                       *types.String
	IsConsistentContainerNameEnabled *types.Bool
	CustomInternalName               *types.String
	NoindexDomains                   *types.List
	InstantDeploy                    *types.Bool
	AutogenerateDomain               *types.Bool
	RedeployOnUpdate                 *types.Bool
	MaxRestartCount                  *types.Int64
}

// applicationCommonModel holds the fields shared by all application resource
// models. Embed this struct to avoid repeating ~60 fields in each model.
type applicationCommonModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ProjectUUID types.String `tfsdk:"project_uuid"`
	ServerUUID  types.String `tfsdk:"server_uuid"`
	// DestinationUUID is create-only. Coolify stores destination_id/destination_type
	// on the morph; GET does not return destination_uuid. Preserve from state.
	DestinationUUID                  types.String   `tfsdk:"destination_uuid"`
	EnvironmentName                  types.String   `tfsdk:"environment_name"`
	PortsExposes                     types.String   `tfsdk:"ports_exposes"`
	Domains                          types.String   `tfsdk:"domains"`
	InstallCommand                   types.String   `tfsdk:"install_command"`
	StartCommand                     types.String   `tfsdk:"start_command"`
	Status                           types.String   `tfsdk:"status"`
	LimitsMemory                     types.String   `tfsdk:"limits_memory"`
	LimitsMemorySwap                 types.String   `tfsdk:"limits_memory_swap"`
	LimitsMemorySwappiness           types.Int64    `tfsdk:"limits_memory_swappiness"`
	LimitsMemoryReservation          types.String   `tfsdk:"limits_memory_reservation"`
	LimitsCPUs                       types.String   `tfsdk:"limits_cpus"`
	LimitsCPUSet                     types.String   `tfsdk:"limits_cpuset"`
	LimitsCPUShares                  types.Int64    `tfsdk:"limits_cpu_shares"`
	HealthCheckEnabled               types.Bool     `tfsdk:"health_check_enabled"`
	HealthCheckPath                  types.String   `tfsdk:"health_check_path"`
	HealthCheckPort                  types.String   `tfsdk:"health_check_port"`
	HealthCheckInterval              types.Int64    `tfsdk:"health_check_interval"`
	HealthCheckTimeout               types.Int64    `tfsdk:"health_check_timeout"`
	HealthCheckRetries               types.Int64    `tfsdk:"health_check_retries"`
	HealthCheckStartPeriod           types.Int64    `tfsdk:"health_check_start_period"`
	HealthCheckCommand               types.String   `tfsdk:"health_check_command"`
	HealthCheckHost                  types.String   `tfsdk:"health_check_host"`
	HealthCheckMethod                types.String   `tfsdk:"health_check_method"`
	HealthCheckResponseText          types.String   `tfsdk:"health_check_response_text"`
	HealthCheckReturnCode            types.Int64    `tfsdk:"health_check_return_code"`
	HealthCheckScheme                types.String   `tfsdk:"health_check_scheme"`
	HealthCheckType                  types.String   `tfsdk:"health_check_type"`
	IsAutoDeployEnabled              types.Bool     `tfsdk:"is_auto_deploy_enabled"`
	BaseDirectory                    types.String   `tfsdk:"base_directory"`
	Dockerfile                       types.String   `tfsdk:"dockerfile"`
	DockerRegistryImageTag           types.String   `tfsdk:"docker_registry_image_tag"`
	DockerComposeDomains             types.String   `tfsdk:"docker_compose_domains"`
	GitCommitSha                     types.String   `tfsdk:"git_commit_sha"`
	PublishDirectory                 types.String   `tfsdk:"publish_directory"`
	WatchPaths                       types.String   `tfsdk:"watch_paths"`
	PreviewURLTemplate               types.String   `tfsdk:"preview_url_template"`
	CustomDockerRunOptions           types.String   `tfsdk:"custom_docker_run_options"`
	CustomLabels                     types.String   `tfsdk:"custom_labels"`
	CustomNetworkAliases             types.String   `tfsdk:"custom_network_aliases"`
	CustomNginxConfiguration         types.String   `tfsdk:"custom_nginx_configuration"`
	PortsMappings                    types.String   `tfsdk:"ports_mappings"`
	ConnectToDockerNetwork           types.Bool     `tfsdk:"connect_to_docker_network"`
	Redirect                         types.String   `tfsdk:"redirect"`
	StaticImage                      types.String   `tfsdk:"static_image"`
	IsStatic                         types.Bool     `tfsdk:"is_static"`
	IsSPA                            types.Bool     `tfsdk:"is_spa"`
	IsForceHTTPSEnabled              types.Bool     `tfsdk:"is_force_https_enabled"`
	IsHTTPBasicAuthEnabled           types.Bool     `tfsdk:"is_http_basic_auth_enabled"`
	HTTPBasicAuthUsername            types.String   `tfsdk:"http_basic_auth_username"`
	HTTPBasicAuthPassword            types.String   `tfsdk:"http_basic_auth_password"`
	PreDeploymentCommand             types.String   `tfsdk:"pre_deployment_command"`
	PreDeploymentCommandContainer    types.String   `tfsdk:"pre_deployment_command_container"`
	PostDeploymentCommand            types.String   `tfsdk:"post_deployment_command"`
	PostDeploymentCommandContainer   types.String   `tfsdk:"post_deployment_command_container"`
	ManualWebhookSecretBitbucket     types.String   `tfsdk:"manual_webhook_secret_bitbucket"`
	ManualWebhookSecretGitea         types.String   `tfsdk:"manual_webhook_secret_gitea"`
	ManualWebhookSecretGitHub        types.String   `tfsdk:"manual_webhook_secret_github"`
	ManualWebhookSecretGitLab        types.String   `tfsdk:"manual_webhook_secret_gitlab"`
	ForceDomainOverride              types.Bool     `tfsdk:"force_domain_override"`
	IsContainerLabelEscapeEnabled    types.Bool     `tfsdk:"is_container_label_escape_enabled"`
	IsPreserveRepositoryEnabled      types.Bool     `tfsdk:"is_preserve_repository_enabled"`
	UseBuildServer                   types.Bool     `tfsdk:"use_build_server"`
	IsPreviewDeploymentsEnabled      types.Bool     `tfsdk:"is_preview_deployments_enabled"`
	UseBuildSecrets                  types.Bool     `tfsdk:"use_build_secrets"`
	StopGracePeriod                  types.Int64    `tfsdk:"stop_grace_period"`
	IsGitSubmodulesEnabled           types.Bool     `tfsdk:"is_git_submodules_enabled"`
	IsGitLfsEnabled                  types.Bool     `tfsdk:"is_git_lfs_enabled"`
	IsGitShallowCloneEnabled         types.Bool     `tfsdk:"is_git_shallow_clone_enabled"`
	DisableBuildCache                types.Bool     `tfsdk:"disable_build_cache"`
	InjectBuildArgsToDockerfile      types.Bool     `tfsdk:"inject_build_args_to_dockerfile"`
	IncludeSourceCommitInBuild       types.Bool     `tfsdk:"include_source_commit_in_build"`
	IsEnvSortingEnabled              types.Bool     `tfsdk:"is_env_sorting_enabled"`
	IsPrDeploymentsPublicEnabled     types.Bool     `tfsdk:"is_pr_deployments_public_enabled"`
	DockerImagesToKeep               types.Int64    `tfsdk:"docker_images_to_keep"`
	IsGzipEnabled                    types.Bool     `tfsdk:"is_gzip_enabled"`
	IsStripprefixEnabled             types.Bool     `tfsdk:"is_stripprefix_enabled"`
	IsRawComposeDeploymentEnabled    types.Bool     `tfsdk:"is_raw_compose_deployment_enabled"`
	IsLogDrainEnabled                types.Bool     `tfsdk:"is_log_drain_enabled"`
	IsGpuEnabled                     types.Bool     `tfsdk:"is_gpu_enabled"`
	GpuDriver                        types.String   `tfsdk:"gpu_driver"`
	GpuCount                         types.String   `tfsdk:"gpu_count"`
	GpuDeviceIds                     types.String   `tfsdk:"gpu_device_ids"`
	GpuOptions                       types.String   `tfsdk:"gpu_options"`
	IsConsistentContainerNameEnabled types.Bool     `tfsdk:"is_consistent_container_name_enabled"`
	CustomInternalName               types.String   `tfsdk:"custom_internal_name"`
	NoindexDomains                   types.List     `tfsdk:"noindex_domains"`
	InstantDeploy                    types.Bool     `tfsdk:"instant_deploy"`
	AutogenerateDomain               types.Bool     `tfsdk:"autogenerate_domain"`
	RedeployOnUpdate                 types.Bool     `tfsdk:"redeploy_on_update"`
	MaxRestartCount                  types.Int64    `tfsdk:"max_restart_count"`
	Timeouts                         timeouts.Value `tfsdk:"timeouts"`
}

// common returns a commonAppFields with pointers to the universal fields.
// Type-specific models call this and then add their own fields.
func (m *applicationCommonModel) common() commonAppFields {
	return commonAppFields{
		UUID: &m.UUID, Name: &m.Name, Description: &m.Description,
		PortsExposes: &m.PortsExposes, Domains: &m.Domains,
		InstallCommand: &m.InstallCommand, StartCommand: &m.StartCommand,
		Status: &m.Status, ProjectUUID: &m.ProjectUUID, ServerUUID: &m.ServerUUID,
		EnvironmentName: &m.EnvironmentName,
		LimitsMemory:    &m.LimitsMemory, LimitsMemorySwap: &m.LimitsMemorySwap,
		LimitsMemorySwappiness: &m.LimitsMemorySwappiness, LimitsMemoryReservation: &m.LimitsMemoryReservation,
		LimitsCPUs: &m.LimitsCPUs, LimitsCPUSet: &m.LimitsCPUSet, LimitsCPUShares: &m.LimitsCPUShares,
		HealthCheckEnabled: &m.HealthCheckEnabled, HealthCheckPath: &m.HealthCheckPath,
		HealthCheckPort: &m.HealthCheckPort, HealthCheckInterval: &m.HealthCheckInterval,
		HealthCheckTimeout: &m.HealthCheckTimeout, HealthCheckRetries: &m.HealthCheckRetries,
		HealthCheckStartPeriod: &m.HealthCheckStartPeriod,
		HealthCheckCommand:     &m.HealthCheckCommand, HealthCheckHost: &m.HealthCheckHost,
		HealthCheckMethod: &m.HealthCheckMethod, HealthCheckResponseText: &m.HealthCheckResponseText,
		HealthCheckReturnCode: &m.HealthCheckReturnCode, HealthCheckScheme: &m.HealthCheckScheme,
		HealthCheckType:     &m.HealthCheckType,
		IsAutoDeployEnabled: &m.IsAutoDeployEnabled,
		BaseDirectory:       &m.BaseDirectory, Dockerfile: &m.Dockerfile,
		DockerRegistryImageTag: &m.DockerRegistryImageTag,
		DockerComposeDomains:   &m.DockerComposeDomains,
		GitCommitSha:           &m.GitCommitSha, PublishDirectory: &m.PublishDirectory,
		WatchPaths: &m.WatchPaths, PreviewURLTemplate: &m.PreviewURLTemplate,
		CustomDockerRunOptions: &m.CustomDockerRunOptions, CustomLabels: &m.CustomLabels,
		CustomNetworkAliases: &m.CustomNetworkAliases, CustomNginxConfiguration: &m.CustomNginxConfiguration,
		PortsMappings: &m.PortsMappings, ConnectToDockerNetwork: &m.ConnectToDockerNetwork,
		Redirect: &m.Redirect, StaticImage: &m.StaticImage,
		IsStatic: &m.IsStatic, IsSPA: &m.IsSPA,
		IsForceHTTPSEnabled: &m.IsForceHTTPSEnabled, IsHTTPBasicAuthEnabled: &m.IsHTTPBasicAuthEnabled,
		HTTPBasicAuthUsername: &m.HTTPBasicAuthUsername, HTTPBasicAuthPassword: &m.HTTPBasicAuthPassword,
		PreDeploymentCommand: &m.PreDeploymentCommand, PreDeploymentCommandContainer: &m.PreDeploymentCommandContainer,
		PostDeploymentCommand: &m.PostDeploymentCommand, PostDeploymentCommandContainer: &m.PostDeploymentCommandContainer,
		ManualWebhookSecretBitbucket: &m.ManualWebhookSecretBitbucket, ManualWebhookSecretGitea: &m.ManualWebhookSecretGitea,
		ManualWebhookSecretGitHub: &m.ManualWebhookSecretGitHub, ManualWebhookSecretGitLab: &m.ManualWebhookSecretGitLab,
		ForceDomainOverride: &m.ForceDomainOverride, IsContainerLabelEscapeEnabled: &m.IsContainerLabelEscapeEnabled,
		IsPreserveRepositoryEnabled: &m.IsPreserveRepositoryEnabled, UseBuildServer: &m.UseBuildServer,
		IsPreviewDeploymentsEnabled: &m.IsPreviewDeploymentsEnabled, UseBuildSecrets: &m.UseBuildSecrets,
		StopGracePeriod:        &m.StopGracePeriod,
		IsGitSubmodulesEnabled: &m.IsGitSubmodulesEnabled, IsGitLfsEnabled: &m.IsGitLfsEnabled,
		IsGitShallowCloneEnabled: &m.IsGitShallowCloneEnabled, DisableBuildCache: &m.DisableBuildCache,
		InjectBuildArgsToDockerfile: &m.InjectBuildArgsToDockerfile, IncludeSourceCommitInBuild: &m.IncludeSourceCommitInBuild,
		IsEnvSortingEnabled: &m.IsEnvSortingEnabled, IsPrDeploymentsPublicEnabled: &m.IsPrDeploymentsPublicEnabled,
		DockerImagesToKeep: &m.DockerImagesToKeep, IsGzipEnabled: &m.IsGzipEnabled,
		IsStripprefixEnabled: &m.IsStripprefixEnabled, IsRawComposeDeploymentEnabled: &m.IsRawComposeDeploymentEnabled,
		IsLogDrainEnabled: &m.IsLogDrainEnabled, IsGpuEnabled: &m.IsGpuEnabled,
		GpuDriver: &m.GpuDriver, GpuCount: &m.GpuCount, GpuDeviceIds: &m.GpuDeviceIds, GpuOptions: &m.GpuOptions,
		IsConsistentContainerNameEnabled: &m.IsConsistentContainerNameEnabled,
		CustomInternalName:               &m.CustomInternalName,
		NoindexDomains:                   &m.NoindexDomains,
		InstantDeploy:                    &m.InstantDeploy, AutogenerateDomain: &m.AutogenerateDomain,
		RedeployOnUpdate: &m.RedeployOnUpdate,
		MaxRestartCount:  &m.MaxRestartCount,
	}
}

// resolveGitRepository reconciles the user's configured git_repository value
// with the API response. Coolify strips "https://github.com/" from GitHub URLs,
// so the API value may differ from the user's input. On import (state is
// null/unknown) the normalized value is used; otherwise the user's value is
// preserved when it matches the raw or normalized API value.
func resolveGitRepository(state types.String, apiValue string) types.String {
	normalized := normalizeGitRepository(apiValue)
	if !state.IsNull() && !state.IsUnknown() {
		sv := state.ValueString()
		if sv == apiValue || sv == normalized || canonicalGitRepo(sv) == canonicalGitRepo(apiValue) {
			return state
		}
	}
	return types.StringValue(normalized)
}

// canonicalGitRepo strips the protocol prefix so that "github.com/org/repo"
// and "https://github.com/org/repo" compare as equal.
func canonicalGitRepo(s string) string {
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	return s
}

// normalizeGitRepository reconstructs a full GitHub URL if the API returned a
// bare org/repo slug. Coolify strips "https://github.com/" from GitHub URLs,
// which causes import state to differ from the user's configured full URL.
func normalizeGitRepository(apiValue string) string {
	if strings.Contains(apiValue, "://") {
		return apiValue
	}
	if strings.HasPrefix(apiValue, "git@") {
		return apiValue
	}
	slashIdx := strings.Index(apiValue, "/")
	dotIdx := strings.Index(apiValue, ".")
	// Dot appears before the first slash: domain prefix (e.g. "github.com/org/repo")
	if dotIdx >= 0 && (slashIdx < 0 || dotIdx < slashIdx) {
		return apiValue
	}
	// Bare slug like "org/repo" or "org/repo.git" (dot after slash is a file extension)
	if slashIdx >= 0 {
		return "https://github.com/" + apiValue
	}
	return apiValue
}

func normalizeCommonAppCreateState(m *applicationCommonModel) {
	flex.NormalizeUnknownString(&m.Name)
	flex.NormalizeUnknownString(&m.Description)
	flex.NormalizeUnknownString(&m.EnvironmentName)
	flex.NormalizeUnknownString(&m.Domains)
	flex.NormalizeUnknownString(&m.Status)
	flex.NormalizeUnknownBool(&m.HealthCheckEnabled)
	flex.NormalizeUnknownBool(&m.IsAutoDeployEnabled)
	flex.NormalizeUnknownString(&m.Redirect)
	flex.NormalizeUnknownString(&m.StaticImage)
	flex.NormalizeUnknownBool(&m.IsStatic)
	flex.NormalizeUnknownBool(&m.IsSPA)
	flex.NormalizeUnknownBool(&m.IsPreserveRepositoryEnabled)
	flex.NormalizeUnknownBool(&m.UseBuildServer)
	flex.NormalizeUnknownBool(&m.IsPreviewDeploymentsEnabled)
	flex.NormalizeUnknownBool(&m.UseBuildSecrets)
	flex.NormalizeUnknownBool(&m.AutogenerateDomain)
	flex.NormalizeUnknownInt64(&m.StopGracePeriod)
	flex.NormalizeUnknownBool(&m.IsGitSubmodulesEnabled)
	flex.NormalizeUnknownBool(&m.IsGitLfsEnabled)
	flex.NormalizeUnknownBool(&m.IsGitShallowCloneEnabled)
	flex.NormalizeUnknownBool(&m.DisableBuildCache)
	flex.NormalizeUnknownBool(&m.InjectBuildArgsToDockerfile)
	flex.NormalizeUnknownBool(&m.IncludeSourceCommitInBuild)
	flex.NormalizeUnknownBool(&m.IsEnvSortingEnabled)
	flex.NormalizeUnknownBool(&m.IsPrDeploymentsPublicEnabled)
	flex.NormalizeUnknownInt64(&m.DockerImagesToKeep)
	flex.NormalizeUnknownBool(&m.IsGzipEnabled)
	flex.NormalizeUnknownBool(&m.IsStripprefixEnabled)
	flex.NormalizeUnknownBool(&m.IsRawComposeDeploymentEnabled)
	flex.NormalizeUnknownBool(&m.IsLogDrainEnabled)
	flex.NormalizeUnknownBool(&m.IsGpuEnabled)
	flex.NormalizeUnknownString(&m.GpuDriver)
	flex.NormalizeUnknownString(&m.GpuCount)
	flex.NormalizeUnknownString(&m.GpuDeviceIds)
	flex.NormalizeUnknownString(&m.GpuOptions)
	flex.NormalizeUnknownBool(&m.IsConsistentContainerNameEnabled)
	flex.NormalizeUnknownString(&m.CustomInternalName)
	// NoindexDomains (List) has no NormalizeUnknown helper; resolve on flatten.
	flex.NormalizeUnknownString(&m.PreviewURLTemplate)
	flex.NormalizeUnknownString(&m.HealthCheckHost)
	flex.NormalizeUnknownString(&m.HealthCheckMethod)
	flex.NormalizeUnknownInt64(&m.HealthCheckReturnCode)
	flex.NormalizeUnknownString(&m.HealthCheckScheme)
	flex.NormalizeUnknownString(&m.HealthCheckType)
	flex.NormalizeUnknownBool(&m.ConnectToDockerNetwork)
	flex.NormalizeUnknownBool(&m.IsForceHTTPSEnabled)
	flex.NormalizeUnknownBool(&m.IsHTTPBasicAuthEnabled)
	flex.NormalizeUnknownString(&m.HTTPBasicAuthUsername)
	flex.NormalizeUnknownString(&m.HTTPBasicAuthPassword)
	flex.NormalizeUnknownString(&m.ManualWebhookSecretBitbucket)
	flex.NormalizeUnknownString(&m.ManualWebhookSecretGitea)
	flex.NormalizeUnknownString(&m.ManualWebhookSecretGitHub)
	flex.NormalizeUnknownString(&m.ManualWebhookSecretGitLab)
	flex.NormalizeUnknownBool(&m.ForceDomainOverride)
	flex.NormalizeUnknownBool(&m.IsContainerLabelEscapeEnabled)
}

// runtimeFieldsChanged returns true if any non-meta/non-immutable field was
// changed between plan and state. When redeploy_on_update is true every
// configuration change (including name, description, webhook secrets, etc.)
// triggers a redeploy so the running container always reflects the latest state.
func runtimeFieldsChanged(plan, state commonAppFields) bool {
	return stringFieldChanged(plan.PortsExposes, state.PortsExposes) ||
		stringFieldChanged(plan.PortsMappings, state.PortsMappings) ||
		stringFieldChanged(plan.Domains, state.Domains) ||
		stringFieldChanged(plan.LimitsMemory, state.LimitsMemory) ||
		stringFieldChanged(plan.LimitsMemorySwap, state.LimitsMemorySwap) ||
		stringFieldChanged(plan.LimitsMemoryReservation, state.LimitsMemoryReservation) ||
		stringFieldChanged(plan.LimitsCPUs, state.LimitsCPUs) ||
		stringFieldChanged(plan.LimitsCPUSet, state.LimitsCPUSet) ||
		int64FieldChanged(plan.LimitsCPUShares, state.LimitsCPUShares) ||
		int64FieldChanged(plan.LimitsMemorySwappiness, state.LimitsMemorySwappiness) ||
		boolFieldChanged(plan.IsForceHTTPSEnabled, state.IsForceHTTPSEnabled) ||
		boolFieldChanged(plan.ConnectToDockerNetwork, state.ConnectToDockerNetwork) ||
		boolFieldChanged(plan.HealthCheckEnabled, state.HealthCheckEnabled) ||
		stringFieldChanged(plan.HealthCheckPath, state.HealthCheckPath) ||
		stringFieldChanged(plan.HealthCheckPort, state.HealthCheckPort) ||
		int64FieldChanged(plan.HealthCheckInterval, state.HealthCheckInterval) ||
		int64FieldChanged(plan.HealthCheckTimeout, state.HealthCheckTimeout) ||
		int64FieldChanged(plan.HealthCheckRetries, state.HealthCheckRetries) ||
		int64FieldChanged(plan.HealthCheckStartPeriod, state.HealthCheckStartPeriod) ||
		stringFieldChanged(plan.HealthCheckCommand, state.HealthCheckCommand) ||
		stringFieldChanged(plan.HealthCheckHost, state.HealthCheckHost) ||
		stringFieldChanged(plan.HealthCheckMethod, state.HealthCheckMethod) ||
		stringFieldChanged(plan.HealthCheckScheme, state.HealthCheckScheme) ||
		stringFieldChanged(plan.HealthCheckResponseText, state.HealthCheckResponseText) ||
		int64FieldChanged(plan.HealthCheckReturnCode, state.HealthCheckReturnCode) ||
		stringFieldChanged(plan.HealthCheckType, state.HealthCheckType) ||
		boolFieldChanged(plan.IsHTTPBasicAuthEnabled, state.IsHTTPBasicAuthEnabled) ||
		stringFieldChanged(plan.HTTPBasicAuthUsername, state.HTTPBasicAuthUsername) ||
		stringFieldChanged(plan.HTTPBasicAuthPassword, state.HTTPBasicAuthPassword) ||
		stringFieldChanged(plan.CustomNetworkAliases, state.CustomNetworkAliases) ||
		dockerComposeDomainsFieldChanged(plan.DockerComposeDomains, state.DockerComposeDomains) ||
		stringFieldChanged(plan.PreDeploymentCommandContainer, state.PreDeploymentCommandContainer) ||
		stringFieldChanged(plan.PostDeploymentCommandContainer, state.PostDeploymentCommandContainer) ||
		stringFieldChanged(plan.CustomLabels, state.CustomLabels) ||
		stringFieldChanged(plan.CustomDockerRunOptions, state.CustomDockerRunOptions) ||
		stringFieldChanged(plan.CustomNginxConfiguration, state.CustomNginxConfiguration) ||
		stringFieldChanged(plan.GitRepository, state.GitRepository) ||
		stringFieldChanged(plan.GitBranch, state.GitBranch) ||
		stringFieldChanged(plan.GitCommitSha, state.GitCommitSha) ||
		stringFieldChanged(plan.DockerfileLocation, state.DockerfileLocation) ||
		stringFieldChanged(plan.Dockerfile, state.Dockerfile) ||
		stringFieldChanged(plan.DockerfileTargetBuild, state.DockerfileTargetBuild) ||
		stringFieldChanged(plan.DockerComposeLocation, state.DockerComposeLocation) ||
		stringFieldChanged(plan.DockerComposeCustomBuildCommand, state.DockerComposeCustomBuildCommand) ||
		stringFieldChanged(plan.DockerComposeCustomStartCommand, state.DockerComposeCustomStartCommand) ||
		stringFieldChanged(plan.BuildPack, state.BuildPack) ||
		stringFieldChanged(plan.BuildCommand, state.BuildCommand) ||
		stringFieldChanged(plan.StartCommand, state.StartCommand) ||
		stringFieldChanged(plan.InstallCommand, state.InstallCommand) ||
		stringFieldChanged(plan.BaseDirectory, state.BaseDirectory) ||
		stringFieldChanged(plan.PublishDirectory, state.PublishDirectory) ||
		stringFieldChanged(plan.PreDeploymentCommand, state.PreDeploymentCommand) ||
		stringFieldChanged(plan.PostDeploymentCommand, state.PostDeploymentCommand) ||
		stringFieldChanged(plan.Redirect, state.Redirect) ||
		stringFieldChanged(plan.StaticImage, state.StaticImage) ||
		boolFieldChanged(plan.IsStatic, state.IsStatic) ||
		boolFieldChanged(plan.IsSPA, state.IsSPA) ||
		stringFieldChanged(plan.WatchPaths, state.WatchPaths) ||
		stringFieldChanged(plan.DockerRegistryImageTag, state.DockerRegistryImageTag) ||
		boolFieldChanged(plan.ForceDomainOverride, state.ForceDomainOverride) ||
		boolFieldChanged(plan.IsContainerLabelEscapeEnabled, state.IsContainerLabelEscapeEnabled) ||
		boolFieldChanged(plan.IsPreserveRepositoryEnabled, state.IsPreserveRepositoryEnabled) ||
		boolFieldChanged(plan.UseBuildServer, state.UseBuildServer) ||
		boolFieldChanged(plan.IsPreviewDeploymentsEnabled, state.IsPreviewDeploymentsEnabled) ||
		boolFieldChanged(plan.UseBuildSecrets, state.UseBuildSecrets) ||
		int64FieldChanged(plan.StopGracePeriod, state.StopGracePeriod) ||
		boolFieldChanged(plan.IsGitSubmodulesEnabled, state.IsGitSubmodulesEnabled) ||
		boolFieldChanged(plan.IsGitLfsEnabled, state.IsGitLfsEnabled) ||
		boolFieldChanged(plan.IsGitShallowCloneEnabled, state.IsGitShallowCloneEnabled) ||
		boolFieldChanged(plan.DisableBuildCache, state.DisableBuildCache) ||
		boolFieldChanged(plan.InjectBuildArgsToDockerfile, state.InjectBuildArgsToDockerfile) ||
		boolFieldChanged(plan.IncludeSourceCommitInBuild, state.IncludeSourceCommitInBuild) ||
		boolFieldChanged(plan.IsEnvSortingEnabled, state.IsEnvSortingEnabled) ||
		boolFieldChanged(plan.IsPrDeploymentsPublicEnabled, state.IsPrDeploymentsPublicEnabled) ||
		int64FieldChanged(plan.DockerImagesToKeep, state.DockerImagesToKeep) ||
		boolFieldChanged(plan.IsGzipEnabled, state.IsGzipEnabled) ||
		boolFieldChanged(plan.IsStripprefixEnabled, state.IsStripprefixEnabled) ||
		boolFieldChanged(plan.IsRawComposeDeploymentEnabled, state.IsRawComposeDeploymentEnabled) ||
		boolFieldChanged(plan.IsLogDrainEnabled, state.IsLogDrainEnabled) ||
		boolFieldChanged(plan.IsGpuEnabled, state.IsGpuEnabled) ||
		stringFieldChanged(plan.GpuDriver, state.GpuDriver) ||
		stringFieldChanged(plan.GpuCount, state.GpuCount) ||
		stringFieldChanged(plan.GpuDeviceIds, state.GpuDeviceIds) ||
		stringFieldChanged(plan.GpuOptions, state.GpuOptions) ||
		boolFieldChanged(plan.IsConsistentContainerNameEnabled, state.IsConsistentContainerNameEnabled) ||
		stringFieldChanged(plan.CustomInternalName, state.CustomInternalName) ||
		listFieldChanged(plan.NoindexDomains, state.NoindexDomains) ||
		stringFieldChanged(plan.Name, state.Name) ||
		stringFieldChanged(plan.Description, state.Description) ||
		boolFieldChanged(plan.IsAutoDeployEnabled, state.IsAutoDeployEnabled) ||
		stringFieldChanged(plan.ManualWebhookSecretBitbucket, state.ManualWebhookSecretBitbucket) ||
		stringFieldChanged(plan.ManualWebhookSecretGitea, state.ManualWebhookSecretGitea) ||
		stringFieldChanged(plan.ManualWebhookSecretGitHub, state.ManualWebhookSecretGitHub) ||
		stringFieldChanged(plan.ManualWebhookSecretGitLab, state.ManualWebhookSecretGitLab)
}

func listFieldChanged(plan, state *types.List) bool {
	if plan == nil || state == nil {
		return false
	}
	return !plan.Equal(*state)
}

func stringFieldChanged(plan, state *types.String) bool {
	if plan == nil || state == nil {
		return false
	}
	return plan.ValueString() != state.ValueString()
}

func int64FieldChanged(plan, state *types.Int64) bool {
	if plan == nil || state == nil {
		return false
	}
	return plan.ValueInt64() != state.ValueInt64()
}

func boolFieldChanged(plan, state *types.Bool) bool {
	if plan == nil || state == nil {
		return false
	}
	return plan.ValueBool() != state.ValueBool()
}

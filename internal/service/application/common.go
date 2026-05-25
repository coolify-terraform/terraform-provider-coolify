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
	InstantDeploy                 *types.Bool
	RedeployOnUpdate              *types.Bool
}

// applicationCommonModel holds the fields shared by all application resource
// models. Embed this struct to avoid repeating ~60 fields in each model.
type applicationCommonModel struct {
	UUID                           types.String   `tfsdk:"uuid"`
	Name                           types.String   `tfsdk:"name"`
	Description                    types.String   `tfsdk:"description"`
	ProjectUUID                    types.String   `tfsdk:"project_uuid"`
	ServerUUID                     types.String   `tfsdk:"server_uuid"`
	EnvironmentName                types.String   `tfsdk:"environment_name"`
	PortsExposes                   types.String   `tfsdk:"ports_exposes"`
	Domains                        types.String   `tfsdk:"domains"`
	InstallCommand                 types.String   `tfsdk:"install_command"`
	StartCommand                   types.String   `tfsdk:"start_command"`
	Status                         types.String   `tfsdk:"status"`
	LimitsMemory                   types.String   `tfsdk:"limits_memory"`
	LimitsMemorySwap               types.String   `tfsdk:"limits_memory_swap"`
	LimitsMemorySwappiness         types.Int64    `tfsdk:"limits_memory_swappiness"`
	LimitsMemoryReservation        types.String   `tfsdk:"limits_memory_reservation"`
	LimitsCPUs                     types.String   `tfsdk:"limits_cpus"`
	LimitsCPUSet                   types.String   `tfsdk:"limits_cpuset"`
	LimitsCPUShares                types.Int64    `tfsdk:"limits_cpu_shares"`
	HealthCheckEnabled             types.Bool     `tfsdk:"health_check_enabled"`
	HealthCheckPath                types.String   `tfsdk:"health_check_path"`
	HealthCheckPort                types.String   `tfsdk:"health_check_port"`
	HealthCheckInterval            types.Int64    `tfsdk:"health_check_interval"`
	HealthCheckTimeout             types.Int64    `tfsdk:"health_check_timeout"`
	HealthCheckRetries             types.Int64    `tfsdk:"health_check_retries"`
	HealthCheckStartPeriod         types.Int64    `tfsdk:"health_check_start_period"`
	HealthCheckCommand             types.String   `tfsdk:"health_check_command"`
	HealthCheckHost                types.String   `tfsdk:"health_check_host"`
	HealthCheckMethod              types.String   `tfsdk:"health_check_method"`
	HealthCheckResponseText        types.String   `tfsdk:"health_check_response_text"`
	HealthCheckReturnCode          types.Int64    `tfsdk:"health_check_return_code"`
	HealthCheckScheme              types.String   `tfsdk:"health_check_scheme"`
	HealthCheckType                types.String   `tfsdk:"health_check_type"`
	IsAutoDeployEnabled            types.Bool     `tfsdk:"is_auto_deploy_enabled"`
	BaseDirectory                  types.String   `tfsdk:"base_directory"`
	Dockerfile                     types.String   `tfsdk:"dockerfile"`
	DockerRegistryImageTag         types.String   `tfsdk:"docker_registry_image_tag"`
	DockerComposeDomains           types.String   `tfsdk:"docker_compose_domains"`
	GitCommitSha                   types.String   `tfsdk:"git_commit_sha"`
	PublishDirectory               types.String   `tfsdk:"publish_directory"`
	WatchPaths                     types.String   `tfsdk:"watch_paths"`
	PreviewURLTemplate             types.String   `tfsdk:"preview_url_template"`
	CustomDockerRunOptions         types.String   `tfsdk:"custom_docker_run_options"`
	CustomLabels                   types.String   `tfsdk:"custom_labels"`
	CustomNetworkAliases           types.String   `tfsdk:"custom_network_aliases"`
	CustomNginxConfiguration       types.String   `tfsdk:"custom_nginx_configuration"`
	PortsMappings                  types.String   `tfsdk:"ports_mappings"`
	ConnectToDockerNetwork         types.Bool     `tfsdk:"connect_to_docker_network"`
	Redirect                       types.String   `tfsdk:"redirect"`
	StaticImage                    types.String   `tfsdk:"static_image"`
	IsStatic                       types.Bool     `tfsdk:"is_static"`
	IsSPA                          types.Bool     `tfsdk:"is_spa"`
	IsForceHTTPSEnabled            types.Bool     `tfsdk:"is_force_https_enabled"`
	IsHTTPBasicAuthEnabled         types.Bool     `tfsdk:"is_http_basic_auth_enabled"`
	HTTPBasicAuthUsername          types.String   `tfsdk:"http_basic_auth_username"`
	HTTPBasicAuthPassword          types.String   `tfsdk:"http_basic_auth_password"`
	PreDeploymentCommand           types.String   `tfsdk:"pre_deployment_command"`
	PreDeploymentCommandContainer  types.String   `tfsdk:"pre_deployment_command_container"`
	PostDeploymentCommand          types.String   `tfsdk:"post_deployment_command"`
	PostDeploymentCommandContainer types.String   `tfsdk:"post_deployment_command_container"`
	ManualWebhookSecretBitbucket   types.String   `tfsdk:"manual_webhook_secret_bitbucket"`
	ManualWebhookSecretGitea       types.String   `tfsdk:"manual_webhook_secret_gitea"`
	ManualWebhookSecretGitHub      types.String   `tfsdk:"manual_webhook_secret_github"`
	ManualWebhookSecretGitLab      types.String   `tfsdk:"manual_webhook_secret_gitlab"`
	ForceDomainOverride            types.Bool     `tfsdk:"force_domain_override"`
	IsContainerLabelEscapeEnabled  types.Bool     `tfsdk:"is_container_label_escape_enabled"`
	IsPreserveRepositoryEnabled    types.Bool     `tfsdk:"is_preserve_repository_enabled"`
	UseBuildServer                 types.Bool     `tfsdk:"use_build_server"`
	InstantDeploy                  types.Bool     `tfsdk:"instant_deploy"`
	RedeployOnUpdate               types.Bool     `tfsdk:"redeploy_on_update"`
	Timeouts                       timeouts.Value `tfsdk:"timeouts"`
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
		InstantDeploy: &m.InstantDeploy, RedeployOnUpdate: &m.RedeployOnUpdate,
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
		stringFieldChanged(plan.DockerComposeDomains, state.DockerComposeDomains) ||
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
		stringFieldChanged(plan.Name, state.Name) ||
		stringFieldChanged(plan.Description, state.Description) ||
		boolFieldChanged(plan.IsAutoDeployEnabled, state.IsAutoDeployEnabled) ||
		stringFieldChanged(plan.ManualWebhookSecretBitbucket, state.ManualWebhookSecretBitbucket) ||
		stringFieldChanged(plan.ManualWebhookSecretGitea, state.ManualWebhookSecretGitea) ||
		stringFieldChanged(plan.ManualWebhookSecretGitHub, state.ManualWebhookSecretGitHub) ||
		stringFieldChanged(plan.ManualWebhookSecretGitLab, state.ManualWebhookSecretGitLab)
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

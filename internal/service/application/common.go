package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	FQDN               *types.String
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
	FQDN                           types.String   `tfsdk:"fqdn"`
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
	Timeouts                       timeouts.Value `tfsdk:"timeouts"`
}

// common returns a commonAppFields with pointers to the universal fields.
// Type-specific models call this and then add their own fields.
func (m *applicationCommonModel) common() commonAppFields {
	return commonAppFields{
		UUID: &m.UUID, Name: &m.Name, Description: &m.Description,
		PortsExposes: &m.PortsExposes, FQDN: &m.FQDN,
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
	}
}

// flattenApplicationCommon maps shared API fields into any application model
// via field pointers. Nil pointers are skipped (allows partial models like
// compose or docker image to omit inapplicable fields).
func flattenApplicationCommon(app *client.Application, f commonAppFields) {
	*f.UUID = types.StringValue(app.UUID)
	*f.Name = types.StringValue(app.Name)
	*f.Description = flex.StringToFramework(app.Description)
	if f.GitRepository != nil {
		*f.GitRepository = resolveGitRepository(*f.GitRepository, app.GitRepository)
	}
	if f.GitBranch != nil {
		*f.GitBranch = types.StringValue(app.GitBranch)
	}
	if f.BuildPack != nil {
		*f.BuildPack = types.StringValue(app.BuildPack)
	}
	// Coolify may override ports_exposes (e.g. return 80 instead of 3000
	// for Dockerfile apps). Preserve the user's configured value.
	if f.PortsExposes != nil && app.PortsExposes != "" {
		if f.PortsExposes.IsNull() || f.PortsExposes.IsUnknown() {
			*f.PortsExposes = types.StringValue(app.PortsExposes)
		}
	}
	*f.FQDN = flex.StringToFramework(app.FQDN)
	// Coolify does not return dockerfile_location on GET for most app types.
	// Preserve the user's configured value to avoid "inconsistent result after apply".
	// The value IS sent on Create/Update, just not returned on read-back.
	if f.DockerfileLocation != nil && app.DockerfileLocation != "" {
		*f.DockerfileLocation = flex.StringToFramework(app.DockerfileLocation)
	}
	if f.InstallCommand != nil {
		*f.InstallCommand = flex.StringToFramework(app.InstallCommand)
	}
	if f.BuildCommand != nil {
		*f.BuildCommand = flex.StringToFramework(app.BuildCommand)
	}
	if f.StartCommand != nil {
		*f.StartCommand = flex.StringToFramework(app.StartCommand)
	}
	*f.Status = flex.StringToFramework(app.Status)
	// Immutable fields: only update if the API returns them (Coolify may
	// omit these from the GET response).
	if app.ProjectUUID != "" {
		*f.ProjectUUID = types.StringValue(app.ProjectUUID)
	}
	if app.ServerUUID != "" {
		*f.ServerUUID = types.StringValue(app.ServerUUID)
	}
	if app.EnvironmentName != "" {
		*f.EnvironmentName = flex.StringToFramework(app.EnvironmentName)
	}
	flattenLimitsAndHealth(app, f)
	flattenExtendedFields(app, f)
}

// flattenLimitsAndHealth sets resource limits, health checks, and auto-deploy
// fields from the API response. Extracted to keep flattenApplicationCommon
// under the gocognit complexity threshold.
func flattenLimitsAndHealth(app *client.Application, f commonAppFields) {
	// Only update optional fields when the user configured them (state is
	// not null/unknown). Coolify returns API defaults ("0", 30, etc.) for
	// unconfigured fields. Setting those would cause "Provider produced
	// inconsistent result after apply" because the plan has null but the
	// read-back would return a value.
	flex.SetStringIfConfigured(f.LimitsMemory, app.LimitsMemory)
	flex.SetStringIfConfigured(f.LimitsMemorySwap, app.LimitsMemorySwap)
	flex.SetStringIfConfigured(f.LimitsMemoryReservation, app.LimitsMemoryReservation)
	flex.SetStringIfConfigured(f.LimitsCPUs, app.LimitsCPUs)
	flex.SetStringIfConfigured(f.LimitsCPUSet, app.LimitsCPUSet)
	flex.SetStringIfConfigured(f.HealthCheckPath, app.HealthCheckPath)
	flex.SetStringIfConfigured(f.HealthCheckPort, app.HealthCheckPort)
	flex.SetInt64IfConfigured(f.LimitsMemorySwappiness, app.LimitsMemorySwappiness)
	flex.SetInt64IfConfigured(f.LimitsCPUShares, app.LimitsCPUShares)
	flex.SetInt64IfConfigured(f.HealthCheckInterval, app.HealthCheckInterval)
	flex.SetInt64IfConfigured(f.HealthCheckTimeout, app.HealthCheckTimeout)
	flex.SetInt64IfConfigured(f.HealthCheckRetries, app.HealthCheckRetries)
	flex.SetInt64IfConfigured(f.HealthCheckStartPeriod, app.HealthCheckStartPeriod)
	// Extended health check fields (optional, no defaults)
	flex.SetStringIfConfigured(f.HealthCheckCommand, app.HealthCheckCommand)
	flex.SetStringIfConfigured(f.HealthCheckResponseText, app.HealthCheckResponseText)
	// Extended health check fields with defaults (always set from API)
	*f.HealthCheckHost = flex.StringValueOrDefault(app.HealthCheckHost, defaultHealthCheckHost)
	*f.HealthCheckMethod = flex.StringValueOrDefault(app.HealthCheckMethod, defaultHealthCheckMeth)
	*f.HealthCheckScheme = flex.StringValueOrDefault(app.HealthCheckScheme, defaultHealthCheckSchm)
	*f.HealthCheckType = flex.StringValueOrDefault(app.HealthCheckType, defaultHealthCheckType)
	if app.HealthCheckReturnCode != nil {
		*f.HealthCheckReturnCode = types.Int64Value(*app.HealthCheckReturnCode)
	}
	// health_check_enabled and is_auto_deploy_enabled are Optional+Computed
	// without Default. Always set them to resolve unknown values after Create.
	// When API returns nil, use the Coolify DB default.
	if app.HealthCheckEnabled != nil {
		*f.HealthCheckEnabled = types.BoolValue(*app.HealthCheckEnabled)
	} else {
		*f.HealthCheckEnabled = types.BoolValue(false)
	}
	if app.IsAutoDeployEnabled != nil {
		*f.IsAutoDeployEnabled = types.BoolValue(*app.IsAutoDeployEnabled)
	}
}

// flattenExtendedFields sets extended application fields from the API response.
// Extracted to keep flattenApplicationCommon under the gocognit complexity threshold.
func flattenExtendedFields(app *client.Application, f commonAppFields) {
	// Optional string fields (set only if user configured them)
	flex.SetStringIfConfigured(f.BaseDirectory, app.BaseDirectory)
	flex.SetStringIfConfigured(f.PublishDirectory, app.PublishDirectory)
	flex.SetStringIfConfigured(f.Dockerfile, app.Dockerfile)
	flex.SetStringIfConfigured(f.DockerRegistryImageTag, app.DockerRegistryImageTag)
	flex.SetStringIfConfigured(f.DockerComposeDomains, app.DockerComposeDomains)
	flex.SetStringIfConfigured(f.GitCommitSha, app.GitCommitSha)
	flex.SetStringIfConfigured(f.WatchPaths, app.WatchPaths)
	flex.SetStringIfConfigured(f.CustomDockerRunOptions, app.CustomDockerRunOptions)
	flex.SetStringIfConfigured(f.CustomLabels, app.CustomLabels)
	flex.SetStringIfConfigured(f.CustomNetworkAliases, app.CustomNetworkAliases)
	flex.SetStringIfConfigured(f.CustomNginxConfiguration, app.CustomNginxConfiguration)
	flex.SetStringIfConfigured(f.PortsMappings, app.PortsMappings)
	flex.SetStringIfConfigured(f.HTTPBasicAuthUsername, app.HTTPBasicAuthUsername)
	flex.SetStringIfConfigured(f.HTTPBasicAuthPassword, app.HTTPBasicAuthPassword)
	flex.SetStringIfConfigured(f.PreDeploymentCommand, app.PreDeploymentCommand)
	flex.SetStringIfConfigured(f.PreDeploymentCommandContainer, app.PreDeploymentCommandContainer)
	flex.SetStringIfConfigured(f.PostDeploymentCommand, app.PostDeploymentCommand)
	flex.SetStringIfConfigured(f.PostDeploymentCommandContainer, app.PostDeploymentCommandContainer)
	// Nil-safe optional string fields (resource-specific extras)
	if f.DockerfileTargetBuild != nil {
		flex.SetStringIfConfigured(f.DockerfileTargetBuild, app.DockerfileTargetBuild)
	}
	if f.DockerComposeLocation != nil {
		flex.SetStringIfConfigured(f.DockerComposeLocation, app.DockerComposeLocation)
	}
	if f.DockerComposeCustomBuildCommand != nil {
		flex.SetStringIfConfigured(f.DockerComposeCustomBuildCommand, app.DockerComposeCustomBuildCommand)
	}
	if f.DockerComposeCustomStartCommand != nil {
		flex.SetStringIfConfigured(f.DockerComposeCustomStartCommand, app.DockerComposeCustomStartCommand)
	}
	flattenExtendedDefaults(app, f)
}

// flattenExtendedDefaults sets fields with Computed+Default and sensitive fields.
func flattenExtendedDefaults(app *client.Application, f commonAppFields) {
	// Computed+Default string fields (always set from API)
	setString := func(dst *types.String, v types.String) {
		if dst == nil {
			return
		}
		*dst = v
	}
	setString(f.Redirect, flex.StringValueOrDefault(app.Redirect, defaultRedirect))
	setString(f.StaticImage, flex.StringValueOrDefault(app.StaticImage, defaultStaticImage))
	// Computed+Default+Sensitive fields (server-generated, always set)
	setString(f.PreviewURLTemplate, flex.StringToFramework(app.PreviewURLTemplate))
	setString(f.ManualWebhookSecretBitbucket, flex.StringToFramework(app.ManualWebhookSecretBitbucket))
	setString(f.ManualWebhookSecretGitea, flex.StringToFramework(app.ManualWebhookSecretGitea))
	setString(f.ManualWebhookSecretGitHub, flex.StringToFramework(app.ManualWebhookSecretGitHub))
	setString(f.ManualWebhookSecretGitLab, flex.StringToFramework(app.ManualWebhookSecretGitLab))
	// Computed+Default bool fields (always set from API)
	setBoolDefault := func(dst *types.Bool, v *bool, def bool) {
		if dst == nil {
			return
		}
		if v != nil {
			*dst = types.BoolValue(*v)
			return
		}
		*dst = types.BoolValue(def)
	}
	setBoolDefault(f.ConnectToDockerNetwork, app.ConnectToDockerNetwork, false)
	setBoolDefault(f.IsHTTPBasicAuthEnabled, app.IsHTTPBasicAuthEnabled, false)
	setBoolDefault(f.IsStatic, app.IsStatic, false)
	setBoolDefault(f.IsSPA, app.IsSPA, false)
	setBoolDefault(f.IsForceHTTPSEnabled, app.IsForceHTTPSEnabled, true)
	setBoolDefault(f.IsContainerLabelEscapeEnabled, app.IsContainerLabelEscapeEnabled, true)
	setBoolDefault(f.IsPreserveRepositoryEnabled, app.IsPreserveRepositoryEnabled, false)
	setBoolDefault(f.UseBuildServer, app.UseBuildServer, false)
	// Optional bool fields (no default)
	if f.ForceDomainOverride != nil && app.ForceDomainOverride != nil {
		if !f.ForceDomainOverride.IsNull() && !f.ForceDomainOverride.IsUnknown() {
			*f.ForceDomainOverride = types.BoolValue(*app.ForceDomainOverride)
		}
	}
}

// buildUpdateInput constructs the shared UpdateApplicationInput from field pointers,
// only including fields that differ between plan and state.
func buildUpdateInput(plan, state commonAppFields) client.UpdateApplicationInput {
	input := buildCoreUpdateFields(plan, state)
	addExtendedUpdateFields(plan, state, &input)
	return input
}

// buildCoreUpdateFields populates the core UpdateApplicationInput fields,
// only including fields that differ between plan and state.
func buildCoreUpdateFields(plan, state commonAppFields) client.UpdateApplicationInput {
	strDiff := flex.StringIfChanged
	int64Diff := flex.Int64IfChanged
	boolDiff := flex.BoolIfChanged
	input := client.UpdateApplicationInput{
		Name:        strDiff(*plan.Name, *state.Name),
		Description: strDiff(*plan.Description, *state.Description),
		FQDN:        strDiff(*plan.FQDN, *state.FQDN),
		// Resource limits
		LimitsMemory:            strDiff(*plan.LimitsMemory, *state.LimitsMemory),
		LimitsMemorySwap:        strDiff(*plan.LimitsMemorySwap, *state.LimitsMemorySwap),
		LimitsMemorySwappiness:  int64Diff(*plan.LimitsMemorySwappiness, *state.LimitsMemorySwappiness),
		LimitsMemoryReservation: strDiff(*plan.LimitsMemoryReservation, *state.LimitsMemoryReservation),
		LimitsCPUs:              strDiff(*plan.LimitsCPUs, *state.LimitsCPUs),
		LimitsCPUSet:            strDiff(*plan.LimitsCPUSet, *state.LimitsCPUSet),
		LimitsCPUShares:         int64Diff(*plan.LimitsCPUShares, *state.LimitsCPUShares),
		// Health checks
		HealthCheckEnabled:      boolDiff(*plan.HealthCheckEnabled, *state.HealthCheckEnabled),
		HealthCheckPath:         strDiff(*plan.HealthCheckPath, *state.HealthCheckPath),
		HealthCheckPort:         strDiff(*plan.HealthCheckPort, *state.HealthCheckPort),
		HealthCheckInterval:     int64Diff(*plan.HealthCheckInterval, *state.HealthCheckInterval),
		HealthCheckTimeout:      int64Diff(*plan.HealthCheckTimeout, *state.HealthCheckTimeout),
		HealthCheckRetries:      int64Diff(*plan.HealthCheckRetries, *state.HealthCheckRetries),
		HealthCheckStartPeriod:  int64Diff(*plan.HealthCheckStartPeriod, *state.HealthCheckStartPeriod),
		HealthCheckCommand:      strDiff(*plan.HealthCheckCommand, *state.HealthCheckCommand),
		HealthCheckHost:         strDiff(*plan.HealthCheckHost, *state.HealthCheckHost),
		HealthCheckMethod:       strDiff(*plan.HealthCheckMethod, *state.HealthCheckMethod),
		HealthCheckResponseText: strDiff(*plan.HealthCheckResponseText, *state.HealthCheckResponseText),
		HealthCheckReturnCode:   int64Diff(*plan.HealthCheckReturnCode, *state.HealthCheckReturnCode),
		HealthCheckScheme:       strDiff(*plan.HealthCheckScheme, *state.HealthCheckScheme),
		HealthCheckType:         strDiff(*plan.HealthCheckType, *state.HealthCheckType),
		// Auto-deploy
		IsAutoDeployEnabled: boolDiff(*plan.IsAutoDeployEnabled, *state.IsAutoDeployEnabled),
	}
	// Nil-safe fields (not present in all resource models)
	if plan.GitRepository != nil && state.GitRepository != nil {
		input.GitRepository = strDiff(*plan.GitRepository, *state.GitRepository)
	}
	if plan.GitBranch != nil && state.GitBranch != nil {
		input.GitBranch = strDiff(*plan.GitBranch, *state.GitBranch)
	}
	if plan.BuildPack != nil && state.BuildPack != nil {
		input.BuildPack = strDiff(*plan.BuildPack, *state.BuildPack)
	}
	if plan.PortsExposes != nil && state.PortsExposes != nil {
		input.PortsExposes = strDiff(*plan.PortsExposes, *state.PortsExposes)
	}
	if plan.InstallCommand != nil && state.InstallCommand != nil {
		input.InstallCommand = strDiff(*plan.InstallCommand, *state.InstallCommand)
	}
	if plan.BuildCommand != nil && state.BuildCommand != nil {
		input.BuildCommand = strDiff(*plan.BuildCommand, *state.BuildCommand)
	}
	if plan.StartCommand != nil && state.StartCommand != nil {
		input.StartCommand = strDiff(*plan.StartCommand, *state.StartCommand)
	}
	if plan.DockerfileLocation != nil && state.DockerfileLocation != nil {
		input.DockerfileLocation = strDiff(*plan.DockerfileLocation, *state.DockerfileLocation)
	}
	return input
}

// addExtendedUpdateFields adds extended fields to an UpdateApplicationInput,
// only including fields that differ between plan and state.
func addExtendedUpdateFields(plan, state commonAppFields, input *client.UpdateApplicationInput) {
	strDiff := flex.StringIfChanged
	boolDiff := flex.BoolIfChanged
	// Build/deploy
	input.BaseDirectory = strDiff(*plan.BaseDirectory, *state.BaseDirectory)
	input.PublishDirectory = strDiff(*plan.PublishDirectory, *state.PublishDirectory)
	input.DockerRegistryImageTag = strDiff(*plan.DockerRegistryImageTag, *state.DockerRegistryImageTag)
	input.DockerComposeDomains = strDiff(*plan.DockerComposeDomains, *state.DockerComposeDomains)
	input.GitCommitSha = strDiff(*plan.GitCommitSha, *state.GitCommitSha)
	input.WatchPaths = strDiff(*plan.WatchPaths, *state.WatchPaths)
	// preview_url_template is not in Coolify v4's update $allowedFields.
	// Container/Network
	input.CustomDockerRunOptions = strDiff(*plan.CustomDockerRunOptions, *state.CustomDockerRunOptions)
	input.CustomLabels = strDiff(*plan.CustomLabels, *state.CustomLabels)
	input.CustomNetworkAliases = strDiff(*plan.CustomNetworkAliases, *state.CustomNetworkAliases)
	input.CustomNginxConfiguration = strDiff(*plan.CustomNginxConfiguration, *state.CustomNginxConfiguration)
	input.PortsMappings = strDiff(*plan.PortsMappings, *state.PortsMappings)
	// Redirect & static
	input.Redirect = strDiff(*plan.Redirect, *state.Redirect)
	input.StaticImage = strDiff(*plan.StaticImage, *state.StaticImage)
	input.IsStatic = boolDiff(*plan.IsStatic, *state.IsStatic)
	input.IsSPA = boolDiff(*plan.IsSPA, *state.IsSPA)
	// Security & Auth
	input.IsForceHTTPSEnabled = boolDiff(*plan.IsForceHTTPSEnabled, *state.IsForceHTTPSEnabled)
	input.IsHTTPBasicAuthEnabled = boolDiff(*plan.IsHTTPBasicAuthEnabled, *state.IsHTTPBasicAuthEnabled)
	input.HTTPBasicAuthUsername = strDiff(*plan.HTTPBasicAuthUsername, *state.HTTPBasicAuthUsername)
	input.HTTPBasicAuthPassword = strDiff(*plan.HTTPBasicAuthPassword, *state.HTTPBasicAuthPassword)
	// Deployment commands
	input.PreDeploymentCommand = strDiff(*plan.PreDeploymentCommand, *state.PreDeploymentCommand)
	input.PreDeploymentCommandContainer = strDiff(*plan.PreDeploymentCommandContainer, *state.PreDeploymentCommandContainer)
	input.PostDeploymentCommand = strDiff(*plan.PostDeploymentCommand, *state.PostDeploymentCommand)
	input.PostDeploymentCommandContainer = strDiff(*plan.PostDeploymentCommandContainer, *state.PostDeploymentCommandContainer)
	// Webhook secrets
	input.ManualWebhookSecretBitbucket = strDiff(*plan.ManualWebhookSecretBitbucket, *state.ManualWebhookSecretBitbucket)
	input.ManualWebhookSecretGitea = strDiff(*plan.ManualWebhookSecretGitea, *state.ManualWebhookSecretGitea)
	input.ManualWebhookSecretGitHub = strDiff(*plan.ManualWebhookSecretGitHub, *state.ManualWebhookSecretGitHub)
	input.ManualWebhookSecretGitLab = strDiff(*plan.ManualWebhookSecretGitLab, *state.ManualWebhookSecretGitLab)
	// Other settings
	input.ConnectToDockerNetwork = boolDiff(*plan.ConnectToDockerNetwork, *state.ConnectToDockerNetwork)
	input.IsContainerLabelEscapeEnabled = boolDiff(*plan.IsContainerLabelEscapeEnabled, *state.IsContainerLabelEscapeEnabled)
	input.IsPreserveRepositoryEnabled = boolDiff(*plan.IsPreserveRepositoryEnabled, *state.IsPreserveRepositoryEnabled)
	input.UseBuildServer = boolDiff(*plan.UseBuildServer, *state.UseBuildServer)
	// Nil-safe resource-specific fields
	if plan.ForceDomainOverride != nil && state.ForceDomainOverride != nil {
		input.ForceDomainOverride = boolDiff(*plan.ForceDomainOverride, *state.ForceDomainOverride)
	}
	if plan.Dockerfile != nil && state.Dockerfile != nil {
		input.Dockerfile = strDiff(*plan.Dockerfile, *state.Dockerfile)
	}
	if plan.DockerfileTargetBuild != nil && state.DockerfileTargetBuild != nil {
		input.DockerfileTargetBuild = strDiff(*plan.DockerfileTargetBuild, *state.DockerfileTargetBuild)
	}
	if plan.DockerComposeLocation != nil && state.DockerComposeLocation != nil {
		input.DockerComposeLocation = strDiff(*plan.DockerComposeLocation, *state.DockerComposeLocation)
	}
	if plan.DockerComposeCustomBuildCommand != nil && state.DockerComposeCustomBuildCommand != nil {
		input.DockerComposeCustomBuildCommand = strDiff(*plan.DockerComposeCustomBuildCommand, *state.DockerComposeCustomBuildCommand)
	}
	if plan.DockerComposeCustomStartCommand != nil && state.DockerComposeCustomStartCommand != nil {
		input.DockerComposeCustomStartCommand = strDiff(*plan.DockerComposeCustomStartCommand, *state.DockerComposeCustomStartCommand)
	}
}

// CommonAppAttrs returns the shared schema attributes for all application types.
func CommonAppAttrs(ctx context.Context, extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := coreAppAttrs(ctx)
	mergeAttrs(attrs, extendedBuildDeployAttrs())
	mergeAttrs(attrs, extendedHealthCheckAttrs())
	mergeAttrs(attrs, securityNetworkAttrs())
	mergeAttrs(attrs, extra)
	return attrs
}

func mergeAttrs(dst, src map[string]schema.Attribute) {
	for k, v := range src {
		dst[k] = v
	}
}

// gitAppAttrs returns the shared schema attributes for Git-backed
// application resources. Keep dockerfile_location scoped here because the
// Dockerfile application resource uses the same attribute name for different
// semantics.
func gitAppAttrs(ctx context.Context, gitRepositoryDescription string, extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := gitAppSourceAttrs(gitRepositoryDescription)
	mergeAttrs(attrs, extra)
	mergeAttrs(attrs, gitAppCommandAttrs())

	return CommonAppAttrs(ctx, attrs)
}

func gitAppSourceAttrs(gitRepositoryDescription string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"git_repository": schema.StringAttribute{
			MarkdownDescription: gitRepositoryDescription,
			Required:            true,
		},
		"git_branch": schema.StringAttribute{
			MarkdownDescription: "The Git branch to deploy (defaults to `main`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("main"),
		},
		"build_pack": schema.StringAttribute{
			MarkdownDescription: "The build pack type. Valid values: `nixpacks`, `dockerfile`, `dockercompose`, `static`.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("nixpacks", "dockerfile", "dockercompose", "static"),
			},
		},
		"ports_exposes": schema.StringAttribute{
			MarkdownDescription: "The ports to expose, as a comma-separated list (e.g. `3000` or `3000,8080`).",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.RegexMatches(regexp.MustCompile(`^\d+(,\d+)*$`), "must be a comma-separated list of port numbers (e.g. \"3000\" or \"3000,8080\")"),
			},
		},
		"dockerfile_location": schema.StringAttribute{
			MarkdownDescription: "The path to the Dockerfile, relative to the repository root.",
			Optional:            true,
		},
	}
}

func gitAppCommandAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"install_command": schema.StringAttribute{
			MarkdownDescription: "The command to run during the install phase.",
			Optional:            true,
		},
		"build_command": schema.StringAttribute{
			MarkdownDescription: "The command to run during the build phase.",
			Optional:            true,
		},
		"start_command": schema.StringAttribute{
			MarkdownDescription: "The command to run to start the application.",
			Optional:            true,
		},
	}
}

// coreAppAttrs returns the core schema attributes (identity, status, limits,
// existing health checks, auto-deploy).
func coreAppAttrs(ctx context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The unique identifier of the application.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the application.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "A description of the application.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"project_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the project this application belongs to. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:          []validator.String{validate.UUID()},
		},
		"server_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the server to deploy the application on. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:          []validator.String{validate.UUID()},
		},
		"environment_name": schema.StringAttribute{
			MarkdownDescription: "The environment name for the application (defaults to `production`). Changing this forces a new resource.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("production"),
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"fqdn": schema.StringAttribute{
			MarkdownDescription: "The fully qualified domain name for the application (must start with http:// or https://).",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Validators:          []validator.String{validate.FQDN()},
		},
		"status": schema.StringAttribute{
			MarkdownDescription: "The current status of the application (e.g. running, stopped, exited). Read-only.",
			Computed:            true,
		},
		// Resource limits
		"limits_memory": schema.StringAttribute{
			MarkdownDescription: "Memory limit (e.g., `512m`, `2g`).",
			Optional:            true,
		},
		"limits_memory_swap": schema.StringAttribute{
			MarkdownDescription: "Memory swap limit (e.g., `1g`).",
			Optional:            true,
		},
		"limits_memory_swappiness": schema.Int64Attribute{
			MarkdownDescription: "Memory swappiness (0-100).",
			Optional:            true,
		},
		"limits_memory_reservation": schema.StringAttribute{
			MarkdownDescription: "Memory reservation (e.g., `256m`).",
			Optional:            true,
		},
		"limits_cpus": schema.StringAttribute{
			MarkdownDescription: "CPU limit (e.g., `0.5`, `2`).",
			Optional:            true,
		},
		"limits_cpuset": schema.StringAttribute{
			MarkdownDescription: "CPU set restriction (e.g., `0-3`, `0,2`).",
			Optional:            true,
		},
		"limits_cpu_shares": schema.Int64Attribute{
			MarkdownDescription: "CPU shares (relative weight).",
			Optional:            true,
		},
		// Health checks
		"health_check_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether health checks are enabled. Coolify defaults to `false` for new applications.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		},
		"health_check_path": schema.StringAttribute{
			MarkdownDescription: "The URL path for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("/"),
		},
		"health_check_port": schema.StringAttribute{
			MarkdownDescription: "The port for health checks.",
			Optional:            true,
		},
		"health_check_interval": schema.Int64Attribute{
			MarkdownDescription: "Health check interval in seconds.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(5),
		},
		"health_check_timeout": schema.Int64Attribute{
			MarkdownDescription: "Health check timeout in seconds.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(5),
		},
		"health_check_retries": schema.Int64Attribute{
			MarkdownDescription: "Number of health check retries.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(10),
		},
		"health_check_start_period": schema.Int64Attribute{
			MarkdownDescription: "Health check start period in seconds.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(5),
		},
		// Auto-deploy
		"is_auto_deploy_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether auto-deploy on push is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
	}
}

// extendedBuildDeployAttrs returns schema attributes for build, deploy, and static settings.
func extendedBuildDeployAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"base_directory": schema.StringAttribute{
			MarkdownDescription: "The base directory for the application source code.",
			Optional:            true,
		},
		"publish_directory": schema.StringAttribute{
			MarkdownDescription: "The directory to publish for static sites.",
			Optional:            true,
		},
		"dockerfile": schema.StringAttribute{
			MarkdownDescription: "Inline Dockerfile content (base64 encoded).",
			Optional:            true,
		},
		"docker_registry_image_tag": schema.StringAttribute{
			MarkdownDescription: "The Docker registry image tag.",
			Optional:            true,
		},
		"docker_compose_domains": schema.StringAttribute{
			MarkdownDescription: "Domain mappings for Docker Compose services.",
			Optional:            true,
		},
		"git_commit_sha": schema.StringAttribute{
			MarkdownDescription: "The specific Git commit SHA to deploy.",
			Optional:            true,
		},
		"watch_paths": schema.StringAttribute{
			MarkdownDescription: "Paths to watch for changes (triggers auto-deploy).",
			Optional:            true,
		},
		"redirect": schema.StringAttribute{
			MarkdownDescription: "Domain redirect mode. Valid values: `www`, `non-www`, `both`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultRedirect),
			Validators:          []validator.String{stringvalidator.OneOf("www", "non-www", "both")},
		},
		"static_image": schema.StringAttribute{
			MarkdownDescription: "The Docker image to use for serving static sites.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultStaticImage),
		},
		"is_static": schema.BoolAttribute{
			MarkdownDescription: "Whether the application is a static site.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"is_spa": schema.BoolAttribute{
			MarkdownDescription: "Whether the application is a single-page application.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"is_preserve_repository_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether to preserve the full Git repository (instead of shallow clone).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"use_build_server": schema.BoolAttribute{
			MarkdownDescription: "Whether to use a build server for building the application.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"preview_url_template": schema.StringAttribute{
			MarkdownDescription: "The URL template for preview deployments. Read-only until Coolify supports setting it on create or update.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"pre_deployment_command": schema.StringAttribute{
			MarkdownDescription: "Command to run before deployment.",
			Optional:            true,
		},
		"pre_deployment_command_container": schema.StringAttribute{
			MarkdownDescription: "Container to run the pre-deployment command in.",
			Optional:            true,
		},
		"post_deployment_command": schema.StringAttribute{
			MarkdownDescription: "Command to run after deployment.",
			Optional:            true,
		},
		"post_deployment_command_container": schema.StringAttribute{
			MarkdownDescription: "Container to run the post-deployment command in.",
			Optional:            true,
		},
	}
}

// extendedHealthCheckAttrs returns schema attributes for extended health check settings.
func extendedHealthCheckAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"health_check_command": schema.StringAttribute{
			MarkdownDescription: "Custom health check command (used when type is `cmd`).",
			Optional:            true,
		},
		"health_check_host": schema.StringAttribute{
			MarkdownDescription: "The host for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultHealthCheckHost),
		},
		"health_check_method": schema.StringAttribute{
			MarkdownDescription: "The HTTP method for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultHealthCheckMeth),
			Validators:          []validator.String{stringvalidator.OneOf("GET", "HEAD", "POST", "OPTIONS")},
		},
		"health_check_response_text": schema.StringAttribute{
			MarkdownDescription: "Expected response text for health check validation.",
			Optional:            true,
		},
		"health_check_return_code": schema.Int64Attribute{
			MarkdownDescription: "Expected HTTP return code for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(defaultHealthCheckCode),
		},
		"health_check_scheme": schema.StringAttribute{
			MarkdownDescription: "The URL scheme for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultHealthCheckSchm),
			Validators:          []validator.String{stringvalidator.OneOf("http", "https")},
		},
		"health_check_type": schema.StringAttribute{
			MarkdownDescription: "The type of health check. Valid values: `http`, `cmd`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultHealthCheckType),
			Validators:          []validator.String{stringvalidator.OneOf("http", "cmd")},
		},
	}
}

// securityNetworkAttrs returns schema attributes for network, security, auth, and webhook settings.
func securityNetworkAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"connect_to_docker_network": schema.BoolAttribute{
			MarkdownDescription: "Whether to connect the application to the Docker network.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"custom_docker_run_options": schema.StringAttribute{
			MarkdownDescription: "Custom Docker run options passed to the container.",
			Optional:            true,
			Validators: []validator.String{
				validate.NoShellMetachars(),
			},
		},
		"custom_labels": schema.StringAttribute{
			MarkdownDescription: "Custom Docker labels for the container, **base64-encoded**. Use `base64encode()` in your configuration.",
			Optional:            true,
		},
		"custom_network_aliases": schema.StringAttribute{
			MarkdownDescription: "Custom network aliases for the container.",
			Optional:            true,
		},
		"custom_nginx_configuration": schema.StringAttribute{
			MarkdownDescription: "Custom Nginx configuration for the application, **base64-encoded**. Use `base64encode()` in your configuration.",
			Optional:            true,
		},
		"ports_mappings": schema.StringAttribute{
			MarkdownDescription: "Port mappings in `host:container` format, comma-separated (e.g. `8080:80` or `8080:80,8443:443`).",
			Optional:            true,
			Validators: []validator.String{
				validate.PortMappings(),
			},
		},
		"force_domain_override": schema.BoolAttribute{
			MarkdownDescription: "Whether to force domain override.",
			Optional:            true,
		},
		"is_container_label_escape_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether container label escaping is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"is_force_https_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether to force HTTPS for the application.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"is_http_basic_auth_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether HTTP Basic Authentication is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"http_basic_auth_username": schema.StringAttribute{
			MarkdownDescription: "Username for HTTP Basic Authentication.",
			Optional:            true,
		},
		"http_basic_auth_password": schema.StringAttribute{
			MarkdownDescription: "Password for HTTP Basic Authentication.",
			Optional:            true,
			Sensitive:           true,
		},
		"manual_webhook_secret_bitbucket": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for Bitbucket.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"manual_webhook_secret_gitea": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for Gitea.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"manual_webhook_secret_github": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for GitHub.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"manual_webhook_secret_gitlab": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for GitLab.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
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
		if sv == apiValue || sv == normalized {
			return state
		}
	}
	return types.StringValue(normalized)
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

// setImportDefaults sets the default values for Computed+Default attributes
// during import. These must be set explicitly because Terraform does not apply
// schema defaults during import; the Read method relies on these initial values
// to avoid null-vs-default conflicts.
func setImportDefaults(ctx context.Context, resp *resource.ImportStateResponse) {
	set := func(attr string, v interface{}) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr), v)...)
	}
	set("health_check_enabled", false)
	set("health_check_path", "/")
	set("health_check_interval", int64(5))
	set("health_check_timeout", int64(5))
	set("health_check_retries", int64(10))
	set("health_check_start_period", int64(5))
	set("is_auto_deploy_enabled", true)
	set("redirect", defaultRedirect)
	set("health_check_type", defaultHealthCheckType)
	set("health_check_method", defaultHealthCheckMeth)
	set("health_check_scheme", defaultHealthCheckSchm)
	set("health_check_return_code", defaultHealthCheckCode)
	set("health_check_host", defaultHealthCheckHost)
	set("static_image", defaultStaticImage)
	set("connect_to_docker_network", false)
	set("is_http_basic_auth_enabled", false)
	set("is_static", false)
	set("is_spa", false)
	set("is_force_https_enabled", true)
	set("is_container_label_escape_enabled", true)
	set("is_preserve_repository_enabled", false)
	set("use_build_server", false)
}

// normalizeUnknown* converts unknown planned values to null before saving
// partial state after create.
func normalizeUnknownString(v *types.String) {
	if v != nil && v.IsUnknown() {
		*v = types.StringNull()
	}
}

func normalizeUnknownBool(v *types.Bool) {
	if v != nil && v.IsUnknown() {
		*v = types.BoolNull()
	}
}

func normalizeUnknownInt64(v *types.Int64) {
	if v != nil && v.IsUnknown() {
		*v = types.Int64Null()
	}
}

func normalizeCommonAppCreateState(m *applicationCommonModel) {
	normalizeUnknownString(&m.Name)
	normalizeUnknownString(&m.Description)
	normalizeUnknownString(&m.EnvironmentName)
	normalizeUnknownString(&m.FQDN)
	normalizeUnknownString(&m.Status)
	normalizeUnknownBool(&m.HealthCheckEnabled)
	normalizeUnknownBool(&m.IsAutoDeployEnabled)
	normalizeUnknownString(&m.Redirect)
	normalizeUnknownString(&m.StaticImage)
	normalizeUnknownBool(&m.IsStatic)
	normalizeUnknownBool(&m.IsSPA)
	normalizeUnknownBool(&m.IsPreserveRepositoryEnabled)
	normalizeUnknownBool(&m.UseBuildServer)
	normalizeUnknownString(&m.PreviewURLTemplate)
	normalizeUnknownString(&m.HealthCheckHost)
	normalizeUnknownString(&m.HealthCheckMethod)
	normalizeUnknownInt64(&m.HealthCheckReturnCode)
	normalizeUnknownString(&m.HealthCheckScheme)
	normalizeUnknownString(&m.HealthCheckType)
	normalizeUnknownBool(&m.ConnectToDockerNetwork)
	normalizeUnknownBool(&m.IsForceHTTPSEnabled)
	normalizeUnknownBool(&m.IsHTTPBasicAuthEnabled)
	normalizeUnknownString(&m.HTTPBasicAuthUsername)
	normalizeUnknownString(&m.HTTPBasicAuthPassword)
	normalizeUnknownString(&m.ManualWebhookSecretBitbucket)
	normalizeUnknownString(&m.ManualWebhookSecretGitea)
	normalizeUnknownString(&m.ManualWebhookSecretGitHub)
	normalizeUnknownString(&m.ManualWebhookSecretGitLab)
	normalizeUnknownBool(&m.ForceDomainOverride)
	normalizeUnknownBool(&m.IsContainerLabelEscapeEnabled)
}

const applicationCreateReadBackFailedSummary = "Application created but refresh failed"

func addApplicationCreateReadBackDiagnostic(resp *resource.CreateResponse, detail string) {
	resp.Diagnostics.AddError(applicationCreateReadBackFailedSummary, detail)
}

func addApplicationCreateReadBackError(resp *resource.CreateResponse, uuid string, err error) {
	addApplicationCreateReadBackDiagnostic(
		resp,
		fmt.Sprintf(
			"Coolify created application %s, but the provider could not read it back: "+
				"Could not read application %s after create: %s. "+
				"The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.",
			uuid,
			uuid,
			err,
		),
	)
}

func addApplicationCreateReadBackNotFoundError(resp *resource.CreateResponse, uuid string) {
	addApplicationCreateReadBackDiagnostic(
		resp,
		fmt.Sprintf(
			"Coolify created application %s, but the provider could not read it back because the API returned 404 on the immediate read-back. "+
				"The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the application becomes readable through the API.",
			uuid,
		),
	)
}

// readBackAfterCreate reads the newly created application. If the immediate
// read-back fails, it leaves the partial state intact and records the failure.
func readBackAfterCreate(ctx context.Context, c *client.Client, uuid string, resp *resource.CreateResponse) *client.Application {
	app, err := c.GetApplication(ctx, uuid)
	if err == nil {
		return app
	}
	if client.IsNotFound(err) {
		addApplicationCreateReadBackNotFoundError(resp, uuid)
		return nil
	}
	addApplicationCreateReadBackError(resp, uuid, err)
	return nil
}

// updateAndReadBack performs the shared update-then-read pattern for all
// application resources.
func updateAndReadBack(
	ctx context.Context,
	c *client.Client,
	uuid string,
	input client.UpdateApplicationInput,
	resp *resource.UpdateResponse,
	flatten func(*client.Application),
) {
	if _, err := c.UpdateApplication(ctx, uuid, input); err != nil {
		resp.Diagnostics.AddError("Error updating application", fmt.Sprintf("application %s: %s", uuid, err))
		return
	}
	app, err := readApplicationAfterUpdate(ctx, c, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error updating application", err.Error())
		return
	}
	flatten(app)
}

func readApplicationAfterUpdate(ctx context.Context, c *client.Client, uuid string) (*client.Application, error) {
	app, err := c.GetApplication(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("reading application %s after update: %w", uuid, err)
	}
	return app, nil
}

// readApplication reads an application by UUID and calls the flatten function.
// If the application is not found, it removes the resource from state.
func readApplication(
	ctx context.Context,
	c *client.Client,
	resourceType string,
	uuid string,
	resp *resource.ReadResponse,
	flatten func(*client.Application),
) {
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
	app, err := c.GetApplication(ctx, uuid)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application", fmt.Sprintf("application %s: %s", uuid, err))
		return
	}
	flatten(app)
}

// deleteApplication deletes an application by UUID and polls until the
// resource is fully removed. Coolify processes application deletions
// asynchronously via DeleteResourceJob; without polling, downstream
// resources (e.g. project) fail to delete because the app still exists.
// A 404 is treated as already-deleted and does not produce an error.
func deleteApplication(
	ctx context.Context,
	c *client.Client,
	resourceType string,
	uuid string,
	resp *resource.DeleteResponse,
) {
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
	if err := c.DeleteApplication(ctx, uuid); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting application", fmt.Sprintf("application %s: %s", uuid, err))
		return
	}
	// Poll until the application is fully removed (up to 2 min).
	// Coolify queues a DeleteResourceJob that tears down containers;
	// on slow hosts this can take well over 60s.
	for range 24 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
		if _, err := c.GetApplication(ctx, uuid); client.IsNotFound(err) {
			return
		}
	}
}

// importApplicationState validates the import ID and sets the initial state
// attributes common to all application resource types.
func importApplicationState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), "production")...)
	setImportDefaults(ctx, resp)
	addApplicationImportSensitiveFieldsWarning(resp)
}

// addApplicationImportSensitiveFieldsWarning explains why imported application
// resources may show diffs for sensitive fields hidden by the API.
func addApplicationImportSensitiveFieldsWarning(resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddWarning(
		"Sensitive fields require token permissions",
		"The Coolify API hides dockerfile, custom_labels, and docker_compose unless the API token has \"root\" or \"read:sensitive\" permission. "+
			"If you see unexpected diffs after import, check your token's permissions in the Coolify dashboard under Security > API Tokens.",
	)
}

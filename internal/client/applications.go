package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Application struct {
	UUID                    string `json:"uuid"`
	Name                    string `json:"name"`
	Description             string `json:"description,omitempty"`
	Domains                 string `json:"fqdn,omitempty"`
	GitRepository           string `json:"git_repository,omitempty"`
	GitBranch               string `json:"git_branch,omitempty"`
	BuildPack               string `json:"build_pack,omitempty"`
	DockerfileLocation      string `json:"dockerfile_location,omitempty"`
	InstallCommand          string `json:"install_command,omitempty"`
	BuildCommand            string `json:"build_command,omitempty"`
	StartCommand            string `json:"start_command,omitempty"`
	PortsExposes            string `json:"ports_exposes,omitempty"`
	ServerUUID              string `json:"server_uuid,omitempty"`
	ProjectUUID             string `json:"project_uuid,omitempty"`
	EnvironmentName         string `json:"environment_name,omitempty"`
	DockerRegistryImageName string `json:"docker_registry_image_name,omitempty"`
	DockerComposeRaw        string `json:"docker_compose_raw,omitempty"`
	Status                  string `json:"status,omitempty"`
	PrivateKeyUUID          string `json:"private_key_uuid,omitempty"`
	GitHubAppUUID           string `json:"github_app_uuid,omitempty"`
	// Resource limits
	LimitsMemory            string `json:"limits_memory,omitempty"`
	LimitsMemorySwap        string `json:"limits_memory_swap,omitempty"`
	LimitsMemorySwappiness  *int64 `json:"limits_memory_swappiness,omitempty"`
	LimitsMemoryReservation string `json:"limits_memory_reservation,omitempty"`
	LimitsCPUs              string `json:"limits_cpus,omitempty"`
	LimitsCPUSet            string `json:"limits_cpuset,omitempty"`
	LimitsCPUShares         *int64 `json:"limits_cpu_shares,omitempty"`
	// Health checks
	HealthCheckEnabled     *bool  `json:"health_check_enabled,omitempty"`
	HealthCheckPath        string `json:"health_check_path,omitempty"`
	HealthCheckPort        string `json:"health_check_port,omitempty"`
	HealthCheckInterval    *int64 `json:"health_check_interval,omitempty"`
	HealthCheckTimeout     *int64 `json:"health_check_timeout,omitempty"`
	HealthCheckRetries     *int64 `json:"health_check_retries,omitempty"`
	HealthCheckStartPeriod *int64 `json:"health_check_start_period,omitempty"`
	// Auto-deploy
	IsAutoDeployEnabled *bool `json:"is_auto_deploy_enabled,omitempty"`
	// Extended build/deploy settings
	BaseDirectory                   string `json:"base_directory,omitempty"`
	PublishDirectory                string `json:"publish_directory,omitempty"`
	Dockerfile                      string `json:"dockerfile,omitempty"`
	DockerfileTargetBuild           string `json:"dockerfile_target_build,omitempty"`
	DockerRegistryImageTag          string `json:"docker_registry_image_tag,omitempty"`
	DockerComposeLocation           string `json:"docker_compose_location,omitempty"`
	DockerComposeCustomBuildCommand string `json:"docker_compose_custom_build_command,omitempty"`
	DockerComposeCustomStartCommand string `json:"docker_compose_custom_start_command,omitempty"`
	// DockerComposeDomains is a JSON string of Coolify's storage map form, or a
	// flexible decoded value; see FlexibleJSONString. Callers should normalize
	// to the write array shape before comparing to Terraform config (#652).
	DockerComposeDomains FlexibleJSONString `json:"docker_compose_domains,omitempty"`
	GitCommitSha         string             `json:"git_commit_sha,omitempty"`
	WatchPaths           string             `json:"watch_paths,omitempty"`
	PreviewURLTemplate   string             `json:"preview_url_template,omitempty"`
	// Container/Network settings
	CustomDockerRunOptions   string `json:"custom_docker_run_options,omitempty"`
	CustomLabels             string `json:"custom_labels,omitempty"`
	CustomNetworkAliases     string `json:"custom_network_aliases,omitempty"`
	CustomNginxConfiguration string `json:"custom_nginx_configuration,omitempty"`
	PortsMappings            string `json:"ports_mappings,omitempty"`
	ConnectToDockerNetwork   *bool  `json:"connect_to_docker_network,omitempty"`
	// Redirect & static
	Redirect    string `json:"redirect,omitempty"`
	StaticImage string `json:"static_image,omitempty"`
	IsStatic    *bool  `json:"is_static,omitempty"`
	IsSPA       *bool  `json:"is_spa,omitempty"`
	// Security & Auth
	IsForceHTTPSEnabled    *bool  `json:"is_force_https_enabled,omitempty"`
	IsHTTPBasicAuthEnabled *bool  `json:"is_http_basic_auth_enabled,omitempty"`
	HTTPBasicAuthUsername  string `json:"http_basic_auth_username,omitempty"`
	HTTPBasicAuthPassword  string `json:"http_basic_auth_password,omitempty"`
	// Extended health checks
	HealthCheckCommand      string `json:"health_check_command,omitempty"`
	HealthCheckHost         string `json:"health_check_host,omitempty"`
	HealthCheckMethod       string `json:"health_check_method,omitempty"`
	HealthCheckResponseText string `json:"health_check_response_text,omitempty"`
	HealthCheckReturnCode   *int64 `json:"health_check_return_code,omitempty"`
	HealthCheckScheme       string `json:"health_check_scheme,omitempty"`
	HealthCheckType         string `json:"health_check_type,omitempty"`
	// Deployment commands
	PreDeploymentCommand           string `json:"pre_deployment_command,omitempty"`
	PreDeploymentCommandContainer  string `json:"pre_deployment_command_container,omitempty"`
	PostDeploymentCommand          string `json:"post_deployment_command,omitempty"`
	PostDeploymentCommandContainer string `json:"post_deployment_command_container,omitempty"`
	// Webhook secrets
	ManualWebhookSecretBitbucket string `json:"manual_webhook_secret_bitbucket,omitempty"`
	ManualWebhookSecretGitea     string `json:"manual_webhook_secret_gitea,omitempty"`
	ManualWebhookSecretGitHub    string `json:"manual_webhook_secret_github,omitempty"`
	ManualWebhookSecretGitLab    string `json:"manual_webhook_secret_gitlab,omitempty"`
	// Other settings
	ForceDomainOverride           *bool `json:"force_domain_override,omitempty"`
	IsContainerLabelEscapeEnabled *bool `json:"is_container_label_escape_enabled,omitempty"`
	IsPreserveRepositoryEnabled   *bool `json:"is_preserve_repository_enabled,omitempty"`
	UseBuildServer                *bool `json:"use_build_server,omitempty"`
	// Application settings (Coolify may nest under settings; also accepted at top-level on write).
	IsPreviewDeploymentsEnabled *bool  `json:"is_preview_deployments_enabled,omitempty"`
	UseBuildSecrets             *bool  `json:"use_build_secrets,omitempty"`
	StopGracePeriod             *int64 `json:"stop_grace_period,omitempty"`
	// Coolify APPLICATION_SETTING_FIELDS (public create/update allow-list).
	IsGitSubmodulesEnabled        *bool  `json:"is_git_submodules_enabled,omitempty"`
	IsGitLfsEnabled               *bool  `json:"is_git_lfs_enabled,omitempty"`
	IsGitShallowCloneEnabled      *bool  `json:"is_git_shallow_clone_enabled,omitempty"`
	DisableBuildCache             *bool  `json:"disable_build_cache,omitempty"`
	InjectBuildArgsToDockerfile   *bool  `json:"inject_build_args_to_dockerfile,omitempty"`
	IncludeSourceCommitInBuild    *bool  `json:"include_source_commit_in_build,omitempty"`
	IsEnvSortingEnabled           *bool  `json:"is_env_sorting_enabled,omitempty"`
	IsPrDeploymentsPublicEnabled  *bool  `json:"is_pr_deployments_public_enabled,omitempty"`
	DockerImagesToKeep            *int64 `json:"docker_images_to_keep,omitempty"`
	IsGzipEnabled                 *bool  `json:"is_gzip_enabled,omitempty"`
	IsStripprefixEnabled          *bool  `json:"is_stripprefix_enabled,omitempty"`
	IsRawComposeDeploymentEnabled *bool  `json:"is_raw_compose_deployment_enabled,omitempty"`
	// Coolify >= v4.3.0 APPLICATION_SETTING_FIELDS additions.
	IsLogDrainEnabled                *bool  `json:"is_log_drain_enabled,omitempty"`
	IsGpuEnabled                     *bool  `json:"is_gpu_enabled,omitempty"`
	GpuDriver                        string `json:"gpu_driver,omitempty"`
	GpuCount                         string `json:"gpu_count,omitempty"`
	GpuDeviceIds                     string `json:"gpu_device_ids,omitempty"`
	GpuOptions                       string `json:"gpu_options,omitempty"`
	IsConsistentContainerNameEnabled *bool  `json:"is_consistent_container_name_enabled,omitempty"`
	CustomInternalName               string `json:"custom_internal_name,omitempty"`
	// NoindexDomains is the subset of domains served with X-Robots-Tag noindex
	// (Coolify >= v4.3.0). JSON array on the wire; empty means none.
	NoindexDomains  []string `json:"noindex_domains,omitempty"`
	MaxRestartCount *int64   `json:"max_restart_count,omitempty"`
	// RestartLimitReached and ContainerPresent are runtime status flags
	// (Coolify tip after 2026-08-31). Pointers so omitted JSON stays null.
	RestartLimitReached *bool `json:"restart_limit_reached,omitempty"`
	ContainerPresent    *bool `json:"container_present,omitempty"`
	// DomainPortOverrides is GET-only (Coolify tip after the Sep 2026
	// applications column). Omitted, null, and Laravel empty-array [] are
	// nil; {} is an empty map. Not on ApplicationsController create/update
	// $allowedFields.
	DomainPortOverrides DomainPortOverridesMap `json:"domain_port_overrides,omitempty"`
	// Nested settings blob from GET responses (promoted after decode).
	Settings *ApplicationSettings `json:"settings,omitempty"`
}

// DomainPortOverridesMap is a GET-only FQDN-to-port map. Laravel's array
// cast JSON-encodes an empty PHP array as []; treat that as omitted.
type DomainPortOverridesMap map[string]int64

// UnmarshalJSON implements json.Unmarshaler.
func (m *DomainPortOverridesMap) UnmarshalJSON(b []byte) error {
	return unmarshalDomainPortOverridesJSON(b, m)
}

func unmarshalDomainPortOverridesJSON(b []byte, dest *DomainPortOverridesMap) error {
	if dest == nil {
		return fmt.Errorf("domain_port_overrides: nil destination")
	}
	b = bytes.TrimSpace(b)
	if string(b) == "null" {
		*dest = nil
		return nil
	}
	if len(b) > 0 && b[0] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(b, &arr); err != nil {
			return fmt.Errorf("domain_port_overrides: %w", err)
		}
		if len(arr) == 0 {
			*dest = nil
			return nil
		}
		return fmt.Errorf("domain_port_overrides: expected object map, got non-empty JSON array")
	}
	var obj map[string]int64
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}
	*dest = obj
	return nil
}

// FlexibleJSONString unmarshals a JSON string or any other JSON value (object,
// array) into its textual form. Coolify stores docker_compose_domains as a
// string column (so GET usually returns a JSON string), but accepting a raw
// object/array avoids hard decode failures if a response is already expanded.
type FlexibleJSONString string

// UnmarshalJSON implements json.Unmarshaler.
func (s *FlexibleJSONString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*s = ""
		return nil
	}
	var asString string
	if err := json.Unmarshal(b, &asString); err == nil {
		*s = FlexibleJSONString(asString)
		return nil
	}
	*s = FlexibleJSONString(b)
	return nil
}

// String returns the underlying value.
func (s FlexibleJSONString) String() string { return string(s) }

// ApplicationSettings holds nested settings from GET application responses.
type ApplicationSettings struct {
	IsPreviewDeploymentsEnabled   *bool  `json:"is_preview_deployments_enabled,omitempty"`
	UseBuildSecrets               *bool  `json:"use_build_secrets,omitempty"`
	StopGracePeriod               *int64 `json:"stop_grace_period,omitempty"`
	IsAutoDeployEnabled           *bool  `json:"is_auto_deploy_enabled,omitempty"`
	UseBuildServer                *bool  `json:"is_build_server_enabled,omitempty"` // settings column name
	IsForceHTTPSEnabled           *bool  `json:"is_force_https_enabled,omitempty"`
	IsGitSubmodulesEnabled        *bool  `json:"is_git_submodules_enabled,omitempty"`
	IsGitLfsEnabled               *bool  `json:"is_git_lfs_enabled,omitempty"`
	IsGitShallowCloneEnabled      *bool  `json:"is_git_shallow_clone_enabled,omitempty"`
	DisableBuildCache             *bool  `json:"disable_build_cache,omitempty"`
	InjectBuildArgsToDockerfile   *bool  `json:"inject_build_args_to_dockerfile,omitempty"`
	IncludeSourceCommitInBuild    *bool  `json:"include_source_commit_in_build,omitempty"`
	IsEnvSortingEnabled           *bool  `json:"is_env_sorting_enabled,omitempty"`
	IsPrDeploymentsPublicEnabled  *bool  `json:"is_pr_deployments_public_enabled,omitempty"`
	DockerImagesToKeep            *int64 `json:"docker_images_to_keep,omitempty"`
	IsGzipEnabled                 *bool  `json:"is_gzip_enabled,omitempty"`
	IsStripprefixEnabled          *bool  `json:"is_stripprefix_enabled,omitempty"`
	IsRawComposeDeploymentEnabled *bool  `json:"is_raw_compose_deployment_enabled,omitempty"`
	// Coolify >= v4.3.0 APPLICATION_SETTING_FIELDS additions.
	IsLogDrainEnabled                *bool  `json:"is_log_drain_enabled,omitempty"`
	IsGpuEnabled                     *bool  `json:"is_gpu_enabled,omitempty"`
	GpuDriver                        string `json:"gpu_driver,omitempty"`
	GpuCount                         string `json:"gpu_count,omitempty"`
	GpuDeviceIds                     string `json:"gpu_device_ids,omitempty"`
	GpuOptions                       string `json:"gpu_options,omitempty"`
	IsConsistentContainerNameEnabled *bool  `json:"is_consistent_container_name_enabled,omitempty"`
	CustomInternalName               string `json:"custom_internal_name,omitempty"`
}

// PromoteSettings copies nested settings onto top-level Application fields when unset.
func (a *Application) PromoteSettings() {
	if a == nil || a.Settings == nil {
		return
	}
	s := a.Settings
	promoteBool(&a.IsPreviewDeploymentsEnabled, s.IsPreviewDeploymentsEnabled)
	promoteBool(&a.UseBuildSecrets, s.UseBuildSecrets)
	promoteInt64(&a.StopGracePeriod, s.StopGracePeriod)
	promoteBool(&a.IsAutoDeployEnabled, s.IsAutoDeployEnabled)
	promoteBool(&a.UseBuildServer, s.UseBuildServer)
	promoteBool(&a.IsForceHTTPSEnabled, s.IsForceHTTPSEnabled)
	promoteBool(&a.IsGitSubmodulesEnabled, s.IsGitSubmodulesEnabled)
	promoteBool(&a.IsGitLfsEnabled, s.IsGitLfsEnabled)
	promoteBool(&a.IsGitShallowCloneEnabled, s.IsGitShallowCloneEnabled)
	promoteBool(&a.DisableBuildCache, s.DisableBuildCache)
	promoteBool(&a.InjectBuildArgsToDockerfile, s.InjectBuildArgsToDockerfile)
	promoteBool(&a.IncludeSourceCommitInBuild, s.IncludeSourceCommitInBuild)
	promoteBool(&a.IsEnvSortingEnabled, s.IsEnvSortingEnabled)
	promoteBool(&a.IsPrDeploymentsPublicEnabled, s.IsPrDeploymentsPublicEnabled)
	promoteInt64(&a.DockerImagesToKeep, s.DockerImagesToKeep)
	promoteBool(&a.IsGzipEnabled, s.IsGzipEnabled)
	promoteBool(&a.IsStripprefixEnabled, s.IsStripprefixEnabled)
	promoteBool(&a.IsRawComposeDeploymentEnabled, s.IsRawComposeDeploymentEnabled)
	promoteBool(&a.IsLogDrainEnabled, s.IsLogDrainEnabled)
	promoteBool(&a.IsGpuEnabled, s.IsGpuEnabled)
	if a.GpuDriver == "" && s.GpuDriver != "" {
		a.GpuDriver = s.GpuDriver
	}
	if a.GpuCount == "" && s.GpuCount != "" {
		a.GpuCount = s.GpuCount
	}
	if a.GpuDeviceIds == "" && s.GpuDeviceIds != "" {
		a.GpuDeviceIds = s.GpuDeviceIds
	}
	if a.GpuOptions == "" && s.GpuOptions != "" {
		a.GpuOptions = s.GpuOptions
	}
	promoteBool(&a.IsConsistentContainerNameEnabled, s.IsConsistentContainerNameEnabled)
	if a.CustomInternalName == "" && s.CustomInternalName != "" {
		a.CustomInternalName = s.CustomInternalName
	}
}

func promoteBool(dst **bool, src *bool) {
	if *dst == nil && src != nil {
		*dst = src
	}
}

func promoteInt64(dst **int64, src *int64) {
	if *dst == nil && src != nil {
		*dst = src
	}
}

type CreatePublicAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
	DestinationUUID    string `json:"destination_uuid,omitempty"`
	EnvironmentName    string `json:"environment_name"`
	EnvironmentUUID    string `json:"environment_uuid,omitempty"`
	GitRepository      string `json:"git_repository"`
	GitBranch          string `json:"git_branch"`
	BuildPack          string `json:"build_pack"`
	PortsExposes       string `json:"ports_exposes"`
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	Domains            string `json:"domains,omitempty"`
	DockerfileLocation string `json:"dockerfile_location,omitempty"`
	InstallCommand     string `json:"install_command,omitempty"`
	BuildCommand       string `json:"build_command,omitempty"`
	StartCommand       string `json:"start_command,omitempty"`
	InstantDeploy      *bool  `json:"instant_deploy,omitempty"`
	AutogenerateDomain *bool  `json:"autogenerate_domain,omitempty"`
}
type UpdateApplicationInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	// Domains uses *string so "" is sent to clear FQDN (omitempty only skips nil
	// pointers; a non-nil pointer to "" is encoded). Coolify maps empty domains
	// to null fqdn when the proxy path applies.
	Domains                 *string `json:"domains,omitempty"`
	GitRepository           *string `json:"git_repository,omitempty"`
	GitBranch               *string `json:"git_branch,omitempty"`
	BuildPack               *string `json:"build_pack,omitempty"`
	DockerfileLocation      *string `json:"dockerfile_location,omitempty"`
	InstallCommand          *string `json:"install_command,omitempty"`
	BuildCommand            *string `json:"build_command,omitempty"`
	StartCommand            *string `json:"start_command,omitempty"`
	PortsExposes            *string `json:"ports_exposes,omitempty"`
	DockerRegistryImageName *string `json:"docker_registry_image_name,omitempty"`
	DockerComposeRaw        *string `json:"docker_compose_raw,omitempty"`
	GitHubAppUUID           *string `json:"github_app_uuid,omitempty"`
	// Resource limits
	LimitsMemory            *string `json:"limits_memory,omitempty"`
	LimitsMemorySwap        *string `json:"limits_memory_swap,omitempty"`
	LimitsMemorySwappiness  *int64  `json:"limits_memory_swappiness,omitempty"`
	LimitsMemoryReservation *string `json:"limits_memory_reservation,omitempty"`
	LimitsCPUs              *string `json:"limits_cpus,omitempty"`
	LimitsCPUSet            *string `json:"limits_cpuset,omitempty"`
	LimitsCPUShares         *int64  `json:"limits_cpu_shares,omitempty"`
	// Health checks
	HealthCheckEnabled     *bool   `json:"health_check_enabled,omitempty"`
	HealthCheckPath        *string `json:"health_check_path,omitempty"`
	HealthCheckPort        *string `json:"health_check_port,omitempty"`
	HealthCheckInterval    *int64  `json:"health_check_interval,omitempty"`
	HealthCheckTimeout     *int64  `json:"health_check_timeout,omitempty"`
	HealthCheckRetries     *int64  `json:"health_check_retries,omitempty"`
	HealthCheckStartPeriod *int64  `json:"health_check_start_period,omitempty"`
	// Auto-deploy
	IsAutoDeployEnabled *bool `json:"is_auto_deploy_enabled,omitempty"`
	// Extended build/deploy settings
	BaseDirectory                   *string `json:"base_directory,omitempty"`
	PublishDirectory                *string `json:"publish_directory,omitempty"`
	Dockerfile                      *string `json:"dockerfile,omitempty"`
	DockerfileTargetBuild           *string `json:"dockerfile_target_build,omitempty"`
	DockerRegistryImageTag          *string `json:"docker_registry_image_tag,omitempty"`
	DockerComposeLocation           *string `json:"docker_compose_location,omitempty"`
	DockerComposeCustomBuildCommand *string `json:"docker_compose_custom_build_command,omitempty"`
	DockerComposeCustomStartCommand *string `json:"docker_compose_custom_start_command,omitempty"`
	// DockerComposeDomains must be a JSON array of {name,domain} on the wire.
	// Coolify validates 'array|nullable' (ApplicationsController). A Go *string
	// would encode as a JSON string and fail with "must be an array" (#652).
	DockerComposeDomains json.RawMessage `json:"docker_compose_domains,omitempty"`
	GitCommitSha         *string         `json:"git_commit_sha,omitempty"`
	WatchPaths           *string         `json:"watch_paths,omitempty"`
	PreviewURLTemplate   *string         `json:"preview_url_template,omitempty"`
	// Container/Network settings
	CustomDockerRunOptions   *string `json:"custom_docker_run_options,omitempty"`
	CustomLabels             *string `json:"custom_labels,omitempty"`
	CustomNetworkAliases     *string `json:"custom_network_aliases,omitempty"`
	CustomNginxConfiguration *string `json:"custom_nginx_configuration,omitempty"`
	PortsMappings            *string `json:"ports_mappings,omitempty"`
	ConnectToDockerNetwork   *bool   `json:"connect_to_docker_network,omitempty"`
	// Redirect & static
	Redirect    *string `json:"redirect,omitempty"`
	StaticImage *string `json:"static_image,omitempty"`
	IsStatic    *bool   `json:"is_static,omitempty"`
	IsSPA       *bool   `json:"is_spa,omitempty"`
	// Security & Auth
	IsForceHTTPSEnabled    *bool   `json:"is_force_https_enabled,omitempty"`
	IsHTTPBasicAuthEnabled *bool   `json:"is_http_basic_auth_enabled,omitempty"`
	HTTPBasicAuthUsername  *string `json:"http_basic_auth_username,omitempty"`
	HTTPBasicAuthPassword  *string `json:"http_basic_auth_password,omitempty"`
	// Extended health checks
	HealthCheckCommand      *string `json:"health_check_command,omitempty"`
	HealthCheckHost         *string `json:"health_check_host,omitempty"`
	HealthCheckMethod       *string `json:"health_check_method,omitempty"`
	HealthCheckResponseText *string `json:"health_check_response_text,omitempty"`
	HealthCheckReturnCode   *int64  `json:"health_check_return_code,omitempty"`
	HealthCheckScheme       *string `json:"health_check_scheme,omitempty"`
	HealthCheckType         *string `json:"health_check_type,omitempty"`
	// Deployment commands
	PreDeploymentCommand           *string `json:"pre_deployment_command,omitempty"`
	PreDeploymentCommandContainer  *string `json:"pre_deployment_command_container,omitempty"`
	PostDeploymentCommand          *string `json:"post_deployment_command,omitempty"`
	PostDeploymentCommandContainer *string `json:"post_deployment_command_container,omitempty"`
	// Webhook secrets
	ManualWebhookSecretBitbucket *string `json:"manual_webhook_secret_bitbucket,omitempty"`
	ManualWebhookSecretGitea     *string `json:"manual_webhook_secret_gitea,omitempty"`
	ManualWebhookSecretGitHub    *string `json:"manual_webhook_secret_github,omitempty"`
	ManualWebhookSecretGitLab    *string `json:"manual_webhook_secret_gitlab,omitempty"`
	// Other settings
	ForceDomainOverride           *bool  `json:"force_domain_override,omitempty"`
	IsContainerLabelEscapeEnabled *bool  `json:"is_container_label_escape_enabled,omitempty"`
	IsPreserveRepositoryEnabled   *bool  `json:"is_preserve_repository_enabled,omitempty"`
	UseBuildServer                *bool  `json:"use_build_server,omitempty"`
	IsPreviewDeploymentsEnabled   *bool  `json:"is_preview_deployments_enabled,omitempty"`
	UseBuildSecrets               *bool  `json:"use_build_secrets,omitempty"`
	StopGracePeriod               *int64 `json:"stop_grace_period,omitempty"`
	// Coolify APPLICATION_SETTING_FIELDS (public create/update allow-list).
	IsGitSubmodulesEnabled        *bool  `json:"is_git_submodules_enabled,omitempty"`
	IsGitLfsEnabled               *bool  `json:"is_git_lfs_enabled,omitempty"`
	IsGitShallowCloneEnabled      *bool  `json:"is_git_shallow_clone_enabled,omitempty"`
	DisableBuildCache             *bool  `json:"disable_build_cache,omitempty"`
	InjectBuildArgsToDockerfile   *bool  `json:"inject_build_args_to_dockerfile,omitempty"`
	IncludeSourceCommitInBuild    *bool  `json:"include_source_commit_in_build,omitempty"`
	IsEnvSortingEnabled           *bool  `json:"is_env_sorting_enabled,omitempty"`
	IsPrDeploymentsPublicEnabled  *bool  `json:"is_pr_deployments_public_enabled,omitempty"`
	DockerImagesToKeep            *int64 `json:"docker_images_to_keep,omitempty"`
	IsGzipEnabled                 *bool  `json:"is_gzip_enabled,omitempty"`
	IsStripprefixEnabled          *bool  `json:"is_stripprefix_enabled,omitempty"`
	IsRawComposeDeploymentEnabled *bool  `json:"is_raw_compose_deployment_enabled,omitempty"`
	// Coolify >= v4.3.0 APPLICATION_SETTING_FIELDS additions.
	IsLogDrainEnabled                *bool   `json:"is_log_drain_enabled,omitempty"`
	IsGpuEnabled                     *bool   `json:"is_gpu_enabled,omitempty"`
	GpuDriver                        *string `json:"gpu_driver,omitempty"`
	GpuCount                         *string `json:"gpu_count,omitempty"`
	GpuDeviceIds                     *string `json:"gpu_device_ids,omitempty"`
	GpuOptions                       *string `json:"gpu_options,omitempty"`
	IsConsistentContainerNameEnabled *bool   `json:"is_consistent_container_name_enabled,omitempty"`
	CustomInternalName               *string `json:"custom_internal_name,omitempty"`
	// NoindexDomains is a JSON array on the wire. Use a non-nil empty slice to
	// clear when the attribute is configured empty (Coolify >= v4.3.0).
	NoindexDomains *[]string `json:"noindex_domains,omitempty"`
	// MaxRestartCount is writable on Coolify >= v4.3.0 (create/update allow list).
	MaxRestartCount *int64 `json:"max_restart_count,omitempty"`
}

func (c *Client) ListApplications(ctx context.Context) ([]Application, error) {
	var a []Application
	if err := c.do(ctx, http.MethodGet, "/api/v1/applications", nil, &a); err != nil {
		return nil, fmt.Errorf("listing applications: %w", err)
	}
	for i := range a {
		a[i].PromoteSettings()
	}
	return a, nil
}

type RestartApplicationResponse struct {
	DeploymentUUID string `json:"deployment_uuid"`
	Message        string `json:"message"`
}

func (c *Client) RestartApplication(ctx context.Context, uuid string) (*RestartApplicationResponse, error) {
	var r RestartApplicationResponse
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/applications/%s/restart", url.PathEscape(uuid)), nil, &r); err != nil {
		return nil, fmt.Errorf("restarting application %s: %w", uuid, err)
	}
	return &r, nil
}
func (c *Client) GetApplication(ctx context.Context, uuid string) (*Application, error) {
	var a Application
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s", url.PathEscape(uuid)), nil, &a); err != nil {
		return nil, fmt.Errorf("getting application %s: %w", uuid, err)
	}
	if a.UUID == "" {
		return nil, fmt.Errorf("getting application %s: API returned empty resource", uuid)
	}
	a.PromoteSettings()
	return &a, nil
}
func (c *Client) CreatePublicApplication(ctx context.Context, input CreatePublicAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/public", input, &a, http.StatusCreated); err != nil {
		if !isMissingDestinationUUID(err) {
			return nil, fmt.Errorf("creating public application: %w", err)
		}
		dest, rerr := c.ResolveDestinationUUID(ctx, input.ServerUUID, input.DestinationUUID)
		if rerr != nil || dest == "" {
			return nil, fmt.Errorf("creating public application: %w", err)
		}
		input.DestinationUUID = dest
		if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/public", input, &a, http.StatusCreated); err != nil {
			return nil, fmt.Errorf("creating public application: %w", err)
		}
	}
	if a.UUID == "" {
		return nil, fmt.Errorf("creating public application: API returned empty UUID")
	}
	return &a, nil
}
func (c *Client) UpdateApplication(ctx context.Context, uuid string, input UpdateApplicationInput) (*Application, error) {
	if !c.SupportsApplicationSettings() {
		if withheld := input.versionGatedWriteKeysPresent(); len(withheld) > 0 {
			tflog.Debug(ctx, "withholding Coolify >= v4.2.0 application write fields unsupported on this instance", map[string]interface{}{
				"uuid":    uuid,
				"version": c.CoolifyVersion,
				"fields":  withheld,
			})
		}
		input.clearApplicationSettings()
	} else if !c.SupportsApplicationSettingsV43() {
		if withheld := input.versionGatedV43WriteKeysPresent(); len(withheld) > 0 {
			tflog.Debug(ctx, "withholding Coolify >= v4.3.0 application write fields unsupported on this instance", map[string]interface{}{
				"uuid":    uuid,
				"version": c.CoolifyVersion,
				"fields":  withheld,
			})
		}
		input.clearApplicationSettingsV43()
	}
	var a Application
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/applications/%s", url.PathEscape(uuid)), input, &a); err != nil {
		return nil, fmt.Errorf("updating application %s: %w", uuid, err)
	}
	return &a, nil
}

// versionGatedWriteKeysPresent returns gated write JSON keys (v4.2.0 and
// v4.3.0) that would appear on the wire for this input before clearing.
func (i UpdateApplicationInput) versionGatedWriteKeysPresent() []string {
	return i.keysPresent(append(append([]string{}, ApplicationSettingsWriteJSONKeys...), ApplicationSettingsV43WriteJSONKeys...))
}

// versionGatedV43WriteKeysPresent returns ApplicationSettingsV43WriteJSONKeys
// that would appear on the wire for this input before clearApplicationSettingsV43.
func (i UpdateApplicationInput) versionGatedV43WriteKeysPresent() []string {
	return i.keysPresent(ApplicationSettingsV43WriteJSONKeys)
}

func (i UpdateApplicationInput) keysPresent(keys []string) []string {
	raw, err := json.Marshal(i)
	if err != nil {
		return nil
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil
	}
	var present []string
	for _, key := range keys {
		if _, ok := body[key]; ok {
			present = append(present, key)
		}
	}
	return present
}

// clearApplicationSettings drops every application write field that Coolify
// only accepts on >= v4.2.0 (and, transitively, the v4.3.0 additions) from the
// payload. That is APPLICATION_SETTING_FIELDS plus is_preview_deployments_enabled
// and use_build_secrets (literals on the same allow list from v4.2.0; absent on
// v4.1.x), plus v4.3.0 fields. The gate lives here, on the single path every
// application PATCH takes, rather than in each caller: a field that cannot be
// written must not depend on which builder assembled the request.
func (i *UpdateApplicationInput) clearApplicationSettings() {
	// v4.2.0 literal allow-list fields (not in APPLICATION_SETTING_FIELDS).
	i.IsPreviewDeploymentsEnabled = nil
	i.UseBuildSecrets = nil
	// v4.2.0 APPLICATION_SETTING_FIELDS.
	i.IsGitSubmodulesEnabled = nil
	i.IsGitLfsEnabled = nil
	i.IsGitShallowCloneEnabled = nil
	i.DisableBuildCache = nil
	i.InjectBuildArgsToDockerfile = nil
	i.IncludeSourceCommitInBuild = nil
	i.IsEnvSortingEnabled = nil
	i.IsPrDeploymentsPublicEnabled = nil
	i.DockerImagesToKeep = nil
	i.IsGzipEnabled = nil
	i.IsStripprefixEnabled = nil
	i.IsRawComposeDeploymentEnabled = nil
	i.StopGracePeriod = nil
	i.clearApplicationSettingsV43()
}

// clearApplicationSettingsV43 drops Coolify >= v4.3.0 application write fields
// (settings additions plus noindex_domains). Used when the instance is
// v4.2.x (settings gate open, v4.3 gate closed).
func (i *UpdateApplicationInput) clearApplicationSettingsV43() {
	i.IsLogDrainEnabled = nil
	i.IsGpuEnabled = nil
	i.GpuDriver = nil
	i.GpuCount = nil
	i.GpuDeviceIds = nil
	i.GpuOptions = nil
	i.IsConsistentContainerNameEnabled = nil
	i.CustomInternalName = nil
	i.NoindexDomains = nil
	i.MaxRestartCount = nil
}

// HasOnlyApplicationSettings reports whether the payload would be empty once
// all version-gated settings fields (v4.2.0 and v4.3.0) are dropped. Callers
// use it to skip a PATCH that the gate would reduce to `{}`.
//
// Counted through the JSON encoding rather than by comparing structs: the input
// carries a json.RawMessage, so it is not comparable, and encoding is what
// decides which fields actually reach Coolify anyway.
func (i UpdateApplicationInput) HasOnlyApplicationSettings() bool {
	stripped := i
	stripped.clearApplicationSettings()
	return i.encodedFieldCount() > 0 && stripped.encodedFieldCount() == 0
}

// HasOnlyApplicationSettingsV43 reports whether the payload would be empty once
// only the v4.3.0 gated fields are dropped. Used to skip a post-create PATCH on
// Coolify 4.2.x when the only extended fields are v4.3.0 additions.
func (i UpdateApplicationInput) HasOnlyApplicationSettingsV43() bool {
	stripped := i
	stripped.clearApplicationSettingsV43()
	return i.encodedFieldCount() > 0 && stripped.encodedFieldCount() == 0
}

// encodedFieldCount returns how many fields the input serialises to.
func (i UpdateApplicationInput) encodedFieldCount() int {
	raw, err := json.Marshal(i)
	if err != nil {
		return 0
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return 0
	}
	return len(body)
}
func (c *Client) DeleteApplication(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/applications/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting application %s: %w", uuid, err)
	}
	return nil
}

func (c *Client) StartApplication(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/applications/%s/start", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("starting application %s: %w", uuid, err)
	}
	return nil
}

func (c *Client) StopApplication(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/applications/%s/stop", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("stopping application %s: %w", uuid, err)
	}
	return nil
}

type CreatePrivateGitAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
	DestinationUUID    string `json:"destination_uuid,omitempty"`
	EnvironmentName    string `json:"environment_name"`
	EnvironmentUUID    string `json:"environment_uuid,omitempty"`
	GitRepository      string `json:"git_repository"`
	GitBranch          string `json:"git_branch"`
	BuildPack          string `json:"build_pack"`
	PortsExposes       string `json:"ports_exposes"`
	PrivateKeyUUID     string `json:"private_key_uuid"`
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	Domains            string `json:"domains,omitempty"`
	DockerfileLocation string `json:"dockerfile_location,omitempty"`
	InstallCommand     string `json:"install_command,omitempty"`
	BuildCommand       string `json:"build_command,omitempty"`
	StartCommand       string `json:"start_command,omitempty"`
	InstantDeploy      *bool  `json:"instant_deploy,omitempty"`
	AutogenerateDomain *bool  `json:"autogenerate_domain,omitempty"`
}

func (c *Client) CreatePrivateGitApplication(ctx context.Context, input CreatePrivateGitAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/private-deploy-key", input, &a, http.StatusCreated); err != nil {
		if !isMissingDestinationUUID(err) {
			return nil, fmt.Errorf("creating private git application: %w", err)
		}
		dest, rerr := c.ResolveDestinationUUID(ctx, input.ServerUUID, input.DestinationUUID)
		if rerr != nil || dest == "" {
			return nil, fmt.Errorf("creating private git application: %w", err)
		}
		input.DestinationUUID = dest
		if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/private-deploy-key", input, &a, http.StatusCreated); err != nil {
			return nil, fmt.Errorf("creating private git application: %w", err)
		}
	}
	if a.UUID == "" {
		return nil, fmt.Errorf("creating private git application: API returned empty UUID")
	}
	return &a, nil
}

type CreateDockerImageAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
	DestinationUUID    string `json:"destination_uuid,omitempty"`
	EnvironmentName    string `json:"environment_name"`
	EnvironmentUUID    string `json:"environment_uuid,omitempty"`
	DockerImage        string `json:"docker_registry_image_name"`
	PortsExposes       string `json:"ports_exposes"`
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	Domains            string `json:"domains,omitempty"`
	InstallCommand     string `json:"install_command,omitempty"`
	StartCommand       string `json:"start_command,omitempty"`
	InstantDeploy      *bool  `json:"instant_deploy,omitempty"`
	AutogenerateDomain *bool  `json:"autogenerate_domain,omitempty"`
}

func (c *Client) CreateDockerImageApplication(ctx context.Context, input CreateDockerImageAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/dockerimage", input, &a, http.StatusCreated); err != nil {
		if !isMissingDestinationUUID(err) {
			return nil, fmt.Errorf("creating docker image application: %w", err)
		}
		dest, rerr := c.ResolveDestinationUUID(ctx, input.ServerUUID, input.DestinationUUID)
		if rerr != nil || dest == "" {
			return nil, fmt.Errorf("creating docker image application: %w", err)
		}
		input.DestinationUUID = dest
		if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/dockerimage", input, &a, http.StatusCreated); err != nil {
			return nil, fmt.Errorf("creating docker image application: %w", err)
		}
	}
	if a.UUID == "" {
		return nil, fmt.Errorf("creating docker image application: API returned empty UUID")
	}
	return &a, nil
}

type CreateDockerfileAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
	DestinationUUID    string `json:"destination_uuid,omitempty"`
	EnvironmentName    string `json:"environment_name"`
	EnvironmentUUID    string `json:"environment_uuid,omitempty"`
	DockerfileLocation string `json:"dockerfile"`
	PortsExposes       string `json:"ports_exposes"`
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	Domains            string `json:"domains,omitempty"`
	InstallCommand     string `json:"install_command,omitempty"`
	BuildCommand       string `json:"build_command,omitempty"`
	StartCommand       string `json:"start_command,omitempty"`
	InstantDeploy      *bool  `json:"instant_deploy,omitempty"`
	AutogenerateDomain *bool  `json:"autogenerate_domain,omitempty"`
}

func (c *Client) CreateDockerfileApplication(ctx context.Context, input CreateDockerfileAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/dockerfile", input, &a, http.StatusCreated); err != nil {
		if !isMissingDestinationUUID(err) {
			return nil, fmt.Errorf("creating dockerfile application: %w", err)
		}
		dest, rerr := c.ResolveDestinationUUID(ctx, input.ServerUUID, input.DestinationUUID)
		if rerr != nil || dest == "" {
			return nil, fmt.Errorf("creating dockerfile application: %w", err)
		}
		input.DestinationUUID = dest
		if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/dockerfile", input, &a, http.StatusCreated); err != nil {
			return nil, fmt.Errorf("creating dockerfile application: %w", err)
		}
	}
	if a.UUID == "" {
		return nil, fmt.Errorf("creating dockerfile application: API returned empty UUID")
	}
	return &a, nil
}

type CreateGitHubAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
	DestinationUUID    string `json:"destination_uuid,omitempty"`
	EnvironmentName    string `json:"environment_name"`
	EnvironmentUUID    string `json:"environment_uuid,omitempty"`
	GitHubAppUUID      string `json:"github_app_uuid"`
	GitRepository      string `json:"git_repository"`
	GitBranch          string `json:"git_branch"`
	BuildPack          string `json:"build_pack"`
	PortsExposes       string `json:"ports_exposes"`
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	Domains            string `json:"domains,omitempty"`
	DockerfileLocation string `json:"dockerfile_location,omitempty"`
	InstallCommand     string `json:"install_command,omitempty"`
	BuildCommand       string `json:"build_command,omitempty"`
	StartCommand       string `json:"start_command,omitempty"`
	InstantDeploy      *bool  `json:"instant_deploy,omitempty"`
	AutogenerateDomain *bool  `json:"autogenerate_domain,omitempty"`
}

func (c *Client) CreateGitHubAppApplication(ctx context.Context, input CreateGitHubAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/private-github-app", input, &a, http.StatusCreated); err != nil {
		if !isMissingDestinationUUID(err) {
			return nil, fmt.Errorf("creating github app application: %w", err)
		}
		dest, rerr := c.ResolveDestinationUUID(ctx, input.ServerUUID, input.DestinationUUID)
		if rerr != nil || dest == "" {
			return nil, fmt.Errorf("creating github app application: %w", err)
		}
		input.DestinationUUID = dest
		if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/private-github-app", input, &a, http.StatusCreated); err != nil {
			return nil, fmt.Errorf("creating github app application: %w", err)
		}
	}
	if a.UUID == "" {
		return nil, fmt.Errorf("creating github app application: API returned empty UUID")
	}
	return &a, nil
}

// ApplicationLog represents a single log line from an application.
type ApplicationLog struct {
	Line      string `json:"line"`
	Timestamp string `json:"timestamp,omitempty"`
}

// GetApplicationLogs returns log lines for an application.
func (c *Client) GetApplicationLogs(ctx context.Context, uuid string) ([]ApplicationLog, error) {
	var logs []ApplicationLog
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s/logs", url.PathEscape(uuid)), nil, &logs); err != nil {
		return nil, fmt.Errorf("getting application logs %s: %w", uuid, err)
	}
	return logs, nil
}

// DeletePreviewDeployment deletes a preview deployment for an application.
func (c *Client) DeletePreviewDeployment(ctx context.Context, appUUID string, pullRequestID int64) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/applications/%s/previews/%d", url.PathEscape(appUUID), pullRequestID), nil, nil); err != nil {
		return fmt.Errorf("deleting preview deployment for application %s pull request %d: %w", appUUID, pullRequestID, err)
	}
	return nil
}

// CreateAppInputJSONTags returns the union of json tags on all application Create*Input structs.
// Used by spectest write coverage for create-only silent-default fields.
func CreateAppInputJSONTags() map[string]struct{} {
	out := map[string]struct{}{}
	for _, tags := range CreateAppInputJSONTagsByType() {
		for name := range tags {
			out[name] = struct{}{}
		}
	}
	return out
}

// CreateAppInputJSONTagsByType returns json tags per Create*AppInput type name.
// Write coverage uses this so a silent-default field cannot be missing from one
// create path while still present on another (union-only checks would miss that).
func CreateAppInputJSONTagsByType() map[string]map[string]struct{} {
	types := map[string]any{
		"CreatePublicAppInput":      CreatePublicAppInput{},
		"CreatePrivateGitAppInput":  CreatePrivateGitAppInput{},
		"CreateDockerImageAppInput": CreateDockerImageAppInput{},
		"CreateDockerfileAppInput":  CreateDockerfileAppInput{},
		"CreateGitHubAppInput":      CreateGitHubAppInput{},
	}
	out := make(map[string]map[string]struct{}, len(types))
	for name, v := range types {
		t := reflect.TypeOf(v)
		tags := map[string]struct{}{}
		for i := 0; i < t.NumField(); i++ {
			tag := t.Field(i).Tag.Get("json")
			if tag == "" || tag == "-" {
				continue
			}
			jname := strings.Split(tag, ",")[0]
			if jname != "" {
				tags[jname] = struct{}{}
			}
		}
		out[name] = tags
	}
	return out
}

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	DockerComposeDomains            string `json:"docker_compose_domains,omitempty"`
	GitCommitSha                    string `json:"git_commit_sha,omitempty"`
	WatchPaths                      string `json:"watch_paths,omitempty"`
	PreviewURLTemplate              string `json:"preview_url_template,omitempty"`
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
}
type CreatePublicAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
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
}
type UpdateApplicationInput struct {
	Name                    *string `json:"name,omitempty"`
	Description             *string `json:"description,omitempty"`
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
	DockerComposeDomains            *string `json:"docker_compose_domains,omitempty"`
	GitCommitSha                    *string `json:"git_commit_sha,omitempty"`
	WatchPaths                      *string `json:"watch_paths,omitempty"`
	PreviewURLTemplate              *string `json:"preview_url_template,omitempty"`
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
	ForceDomainOverride           *bool `json:"force_domain_override,omitempty"`
	IsContainerLabelEscapeEnabled *bool `json:"is_container_label_escape_enabled,omitempty"`
	IsPreserveRepositoryEnabled   *bool `json:"is_preserve_repository_enabled,omitempty"`
	UseBuildServer                *bool `json:"use_build_server,omitempty"`
}

func (c *Client) ListApplications(ctx context.Context) ([]Application, error) {
	var a []Application
	if err := c.do(ctx, http.MethodGet, "/api/v1/applications", nil, &a); err != nil {
		return nil, fmt.Errorf("listing applications: %w", err)
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
	return &a, nil
}
func (c *Client) CreatePublicApplication(ctx context.Context, input CreatePublicAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/public", input, &a, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating public application: %w", err)
	}
	if a.UUID == "" {
		return nil, fmt.Errorf("creating public application: API returned empty UUID")
	}
	return &a, nil
}
func (c *Client) UpdateApplication(ctx context.Context, uuid string, input UpdateApplicationInput) (*Application, error) {
	var a Application
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/applications/%s", url.PathEscape(uuid)), input, &a); err != nil {
		return nil, fmt.Errorf("updating application %s: %w", uuid, err)
	}
	return &a, nil
}
func (c *Client) DeleteApplication(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/applications/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting application %s: %w", uuid, err)
	}
	return nil
}

func (c *Client) StartApplication(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s/start", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("starting application %s: %w", uuid, err)
	}
	return nil
}

func (c *Client) StopApplication(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s/stop", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("stopping application %s: %w", uuid, err)
	}
	return nil
}

type CreatePrivateGitAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
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
}

func (c *Client) CreatePrivateGitApplication(ctx context.Context, input CreatePrivateGitAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/private-deploy-key", input, &a, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating private git application: %w", err)
	}
	if a.UUID == "" {
		return nil, fmt.Errorf("creating private git application: API returned empty UUID")
	}
	return &a, nil
}

type CreateDockerImageAppInput struct {
	ProjectUUID     string `json:"project_uuid"`
	ServerUUID      string `json:"server_uuid"`
	EnvironmentName string `json:"environment_name"`
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
	DockerImage     string `json:"docker_registry_image_name"`
	PortsExposes    string `json:"ports_exposes"`
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	Domains         string `json:"domains,omitempty"`
	InstallCommand  string `json:"install_command,omitempty"`
	StartCommand    string `json:"start_command,omitempty"`
}

func (c *Client) CreateDockerImageApplication(ctx context.Context, input CreateDockerImageAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/dockerimage", input, &a, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating docker image application: %w", err)
	}
	if a.UUID == "" {
		return nil, fmt.Errorf("creating docker image application: API returned empty UUID")
	}
	return &a, nil
}

type CreateDockerfileAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
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
}

func (c *Client) CreateDockerfileApplication(ctx context.Context, input CreateDockerfileAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/dockerfile", input, &a, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating dockerfile application: %w", err)
	}
	if a.UUID == "" {
		return nil, fmt.Errorf("creating dockerfile application: API returned empty UUID")
	}
	return &a, nil
}

type CreateGitHubAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
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
}

func (c *Client) CreateGitHubAppApplication(ctx context.Context, input CreateGitHubAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/private-github-app", input, &a, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating github app application: %w", err)
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

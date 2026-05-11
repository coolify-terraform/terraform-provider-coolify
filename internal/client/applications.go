package client

import (
	"context"
	"fmt"
	"net/http"
)

type Application struct {
	UUID                    string `json:"uuid"`
	Name                    string `json:"name"`
	Description             string `json:"description,omitempty"`
	FQDN                    string `json:"fqdn,omitempty"`
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
	DockerRegistryImageName string `json:"docker_registry_image_name,omitempty"`
	DockerComposeRaw        string `json:"docker_compose_raw,omitempty"`
	Status                  string `json:"status,omitempty"`
}
type CreatePublicAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
	EnvironmentName    string `json:"environment_name"`
	GitRepository      string `json:"git_repository"`
	GitBranch          string `json:"git_branch"`
	BuildPack          string `json:"build_pack"`
	PortsExposes       string `json:"ports_exposes"`
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	FQDN               string `json:"fqdn,omitempty"`
	DockerfileLocation string `json:"dockerfile_location,omitempty"`
	InstallCommand     string `json:"install_command,omitempty"`
	BuildCommand       string `json:"build_command,omitempty"`
	StartCommand       string `json:"start_command,omitempty"`
}
type UpdateApplicationInput struct {
	Name                    *string `json:"name,omitempty"`
	Description             *string `json:"description,omitempty"`
	FQDN                    *string `json:"fqdn,omitempty"`
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
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/applications/%s/restart", uuid), nil, &r); err != nil {
		return nil, fmt.Errorf("restarting application %s: %w", uuid, err)
	}
	return &r, nil
}
func (c *Client) GetApplication(ctx context.Context, uuid string) (*Application, error) {
	var a Application
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s", uuid), nil, &a); err != nil {
		return nil, fmt.Errorf("getting application %s: %w", uuid, err)
	}
	return &a, nil
}
func (c *Client) CreatePublicApplication(ctx context.Context, input CreatePublicAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/public", input, &a, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating public application: %w", err)
	}
	return &a, nil
}
func (c *Client) UpdateApplication(ctx context.Context, uuid string, input UpdateApplicationInput) (*Application, error) {
	var a Application
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/applications/%s", uuid), input, &a); err != nil {
		return nil, fmt.Errorf("updating application %s: %w", uuid, err)
	}
	return &a, nil
}
func (c *Client) DeleteApplication(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/applications/%s", uuid), nil, nil); err != nil {
		return fmt.Errorf("deleting application %s: %w", uuid, err)
	}
	return nil
}

func (c *Client) StartApplication(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s/start", uuid), nil, nil); err != nil {
		return fmt.Errorf("starting application %s: %w", uuid, err)
	}
	return nil
}

func (c *Client) StopApplication(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s/stop", uuid), nil, nil); err != nil {
		return fmt.Errorf("stopping application %s: %w", uuid, err)
	}
	return nil
}

type CreatePrivateGitAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
	EnvironmentName    string `json:"environment_name"`
	GitRepository      string `json:"git_repository"`
	GitBranch          string `json:"git_branch"`
	BuildPack          string `json:"build_pack"`
	PortsExposes       string `json:"ports_exposes"`
	PrivateKeyUUID     string `json:"private_key_uuid"`
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	FQDN               string `json:"fqdn,omitempty"`
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
	return &a, nil
}

type CreateDockerComposeAppInput struct {
	ProjectUUID      string `json:"project_uuid"`
	ServerUUID       string `json:"server_uuid"`
	EnvironmentName  string `json:"environment_name"`
	DockerComposeRaw string `json:"docker_compose_raw"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	FQDN             string `json:"fqdn,omitempty"`
	InstantDeploy    bool   `json:"instant_deploy,omitempty"`
}

func (c *Client) CreateDockerComposeApplication(ctx context.Context, input CreateDockerComposeAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/dockercompose", input, &a, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating docker compose application: %w", err)
	}
	return &a, nil
}

type CreateDockerImageAppInput struct {
	ProjectUUID     string `json:"project_uuid"`
	ServerUUID      string `json:"server_uuid"`
	EnvironmentName string `json:"environment_name"`
	DockerImage     string `json:"docker_registry_image_name"`
	PortsExposes    string `json:"ports_exposes"`
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	FQDN            string `json:"fqdn,omitempty"`
	InstallCommand  string `json:"install_command,omitempty"`
	StartCommand    string `json:"start_command,omitempty"`
}

func (c *Client) CreateDockerImageApplication(ctx context.Context, input CreateDockerImageAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/dockerimage", input, &a, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating docker image application: %w", err)
	}
	return &a, nil
}

type CreateDockerfileAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
	EnvironmentName    string `json:"environment_name"`
	DockerfileLocation string `json:"dockerfile_location"`
	PortsExposes       string `json:"ports_exposes"`
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	FQDN               string `json:"fqdn,omitempty"`
	InstallCommand     string `json:"install_command,omitempty"`
	BuildCommand       string `json:"build_command,omitempty"`
	StartCommand       string `json:"start_command,omitempty"`
}

func (c *Client) CreateDockerfileApplication(ctx context.Context, input CreateDockerfileAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/dockerfile", input, &a, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating dockerfile application: %w", err)
	}
	return &a, nil
}

type CreateGitHubAppInput struct {
	ProjectUUID        string `json:"project_uuid"`
	ServerUUID         string `json:"server_uuid"`
	EnvironmentName    string `json:"environment_name"`
	GitHubAppUUID      string `json:"github_app_uuid"`
	GitRepository      string `json:"git_repository"`
	GitBranch          string `json:"git_branch"`
	BuildPack          string `json:"build_pack"`
	PortsExposes       string `json:"ports_exposes"`
	Name               string `json:"name,omitempty"`
	Description        string `json:"description,omitempty"`
	FQDN               string `json:"fqdn,omitempty"`
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
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s/logs", uuid), nil, &logs); err != nil {
		return nil, fmt.Errorf("getting application logs %s: %w", uuid, err)
	}
	return logs, nil
}

// DeletePreviewDeployment deletes a preview deployment for an application.
func (c *Client) DeletePreviewDeployment(ctx context.Context, appUUID string, pullRequestID int64) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/applications/%s/previews/%d", appUUID, pullRequestID), nil, nil); err != nil {
		return fmt.Errorf("deleting preview deployment for application %s pull request %d: %w", appUUID, pullRequestID, err)
	}
	return nil
}

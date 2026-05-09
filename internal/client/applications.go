package client

import (
	"context"
	"fmt"
	"net/http"
)

type Application struct {
	UUID               string `json:"uuid"`
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	FQDN               string `json:"fqdn,omitempty"`
	GitRepository      string `json:"git_repository,omitempty"`
	GitBranch          string `json:"git_branch,omitempty"`
	BuildPack          string `json:"build_pack,omitempty"`
	DockerfileLocation string `json:"dockerfile_location,omitempty"`
	InstallCommand     string `json:"install_command,omitempty"`
	BuildCommand       string `json:"build_command,omitempty"`
	StartCommand       string `json:"start_command,omitempty"`
	PortsExposes       string `json:"ports_exposes,omitempty"`
	ServerUUID         string `json:"server_uuid,omitempty"`
	ProjectUUID        string `json:"project_uuid,omitempty"`
	Status             string `json:"status,omitempty"`
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
	Name               *string `json:"name,omitempty"`
	Description        *string `json:"description,omitempty"`
	FQDN               *string `json:"fqdn,omitempty"`
	GitRepository      *string `json:"git_repository,omitempty"`
	GitBranch          *string `json:"git_branch,omitempty"`
	BuildPack          *string `json:"build_pack,omitempty"`
	DockerfileLocation *string `json:"dockerfile_location,omitempty"`
	InstallCommand     *string `json:"install_command,omitempty"`
	BuildCommand       *string `json:"build_command,omitempty"`
	StartCommand       *string `json:"start_command,omitempty"`
	PortsExposes       *string `json:"ports_exposes,omitempty"`
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

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

func (c *Client) GetApplication(ctx context.Context, uuid string) (*Application, error) {
	var a Application
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s", uuid), nil, &a); err != nil {
		return nil, err
	}
	return &a, nil
}
func (c *Client) CreatePublicApplication(ctx context.Context, input CreatePublicAppInput) (*Application, error) {
	var a Application
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/applications/public", input, &a, http.StatusCreated); err != nil {
		return nil, err
	}
	return &a, nil
}
func (c *Client) UpdateApplication(ctx context.Context, uuid string, input UpdateApplicationInput) (*Application, error) {
	var a Application
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/applications/%s", uuid), input, &a); err != nil {
		return nil, err
	}
	return &a, nil
}
func (c *Client) DeleteApplication(ctx context.Context, uuid string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/applications/%s", uuid), nil, nil)
}

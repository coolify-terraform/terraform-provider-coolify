package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ServiceApplication represents a container within a service (from the
// applications relation loaded by GET /services/{uuid}).
type ServiceApplication struct {
	Name string `json:"name"`
	FQDN string `json:"fqdn,omitempty"`
}

type Service struct {
	UUID                          string               `json:"uuid"`
	Name                          string               `json:"name"`
	Description                   string               `json:"description,omitempty"`
	Type                          string               `json:"type"`
	ServerUUID                    string               `json:"server_uuid,omitempty"`
	ProjectUUID                   string               `json:"project_uuid,omitempty"`
	EnvironmentName               string               `json:"environment_name,omitempty"`
	Status                        string               `json:"status,omitempty"`
	DockerCompose                 string               `json:"docker_compose,omitempty"`
	DockerComposeRaw              string               `json:"docker_compose_raw,omitempty"`
	ConnectToNetwork              *bool                `json:"connect_to_docker_network,omitempty"`
	IsContainerLabelEscapeEnabled *bool                `json:"is_container_label_escape_enabled,omitempty"`
	ConfigHash                    string               `json:"config_hash,omitempty"`
	Applications                  []ServiceApplication `json:"applications,omitempty"`
}

// ServiceURL maps a compose service name to one or more URLs.
type ServiceURL struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
type CreateServiceInput struct {
	Type                string       `json:"type,omitempty"`
	Name                string       `json:"name,omitempty"`
	Description         string       `json:"description,omitempty"`
	ServerUUID          string       `json:"server_uuid"`
	ProjectUUID         string       `json:"project_uuid"`
	EnvironmentName     string       `json:"environment_name"`
	EnvironmentUUID     string       `json:"environment_uuid,omitempty"`
	InstantDeploy       *bool        `json:"instant_deploy,omitempty"`
	DockerComposeRaw    *string      `json:"docker_compose_raw,omitempty"`
	URLs                []ServiceURL `json:"urls,omitempty"`
	ForceDomainOverride *bool        `json:"force_domain_override,omitempty"`
}

func (c *Client) ListServices(ctx context.Context) ([]Service, error) {
	var s []Service
	if err := c.do(ctx, http.MethodGet, "/api/v1/services", nil, &s); err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}
	return s, nil
}
func (c *Client) GetService(ctx context.Context, uuid string) (*Service, error) {
	var s Service
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/services/%s", url.PathEscape(uuid)), nil, &s); err != nil {
		return nil, fmt.Errorf("getting service %s: %w", uuid, err)
	}
	return &s, nil
}
func (c *Client) CreateService(ctx context.Context, input CreateServiceInput) (*Service, error) {
	var s Service
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/services", input, &s, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating service: %w", err)
	}
	if s.UUID == "" {
		return nil, fmt.Errorf("creating service: API returned empty UUID")
	}
	return &s, nil
}

type UpdateServiceInput struct {
	Name                          *string      `json:"name,omitempty"`
	Description                   *string      `json:"description,omitempty"`
	DockerComposeRaw              *string      `json:"docker_compose_raw,omitempty"`
	ConnectToNetwork              *bool        `json:"connect_to_docker_network,omitempty"`
	IsContainerLabelEscapeEnabled *bool        `json:"is_container_label_escape_enabled,omitempty"`
	InstantDeploy                 *bool        `json:"instant_deploy,omitempty"`
	URLs                          []ServiceURL `json:"urls,omitempty"`
	ForceDomainOverride           *bool        `json:"force_domain_override,omitempty"`
}

func (c *Client) UpdateService(ctx context.Context, uuid string, input UpdateServiceInput) (*Service, error) {
	var s Service
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/services/%s", url.PathEscape(uuid)), input, &s); err != nil {
		return nil, fmt.Errorf("updating service %s: %w", uuid, err)
	}
	return &s, nil
}
func (c *Client) DeleteService(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/services/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting service %s: %w", uuid, err)
	}
	return nil
}
func (c *Client) StartService(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/services/%s/start", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("starting service %s: %w", uuid, err)
	}
	return nil
}
func (c *Client) StopService(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/services/%s/stop", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("stopping service %s: %w", uuid, err)
	}
	return nil
}

// RestartService restarts a service.
func (c *Client) RestartService(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/services/%s/restart", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("restarting service %s: %w", uuid, err)
	}
	return nil
}

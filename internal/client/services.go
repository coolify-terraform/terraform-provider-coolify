package client

import (
	"context"
	"fmt"
	"net/http"
)

type Service struct {
	UUID            string `json:"uuid"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	Type            string `json:"type"`
	ServerUUID      string `json:"server_uuid,omitempty"`
	ProjectUUID     string `json:"project_uuid,omitempty"`
	EnvironmentName string `json:"environment_name,omitempty"`
}
type CreateServiceInput struct {
	Type            string `json:"type"`
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	ServerUUID      string `json:"server_uuid"`
	ProjectUUID     string `json:"project_uuid"`
	EnvironmentName string `json:"environment_name"`
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
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
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/services/%s", uuid), nil, &s); err != nil {
		return nil, fmt.Errorf("getting service %s: %w", uuid, err)
	}
	return &s, nil
}
func (c *Client) CreateService(ctx context.Context, input CreateServiceInput) (*Service, error) {
	var s Service
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/services", input, &s, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating service: %w", err)
	}
	return &s, nil
}

type UpdateServiceInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (c *Client) UpdateService(ctx context.Context, uuid string, input UpdateServiceInput) (*Service, error) {
	var s Service
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/services/%s", uuid), input, &s); err != nil {
		return nil, fmt.Errorf("updating service %s: %w", uuid, err)
	}
	return &s, nil
}
func (c *Client) DeleteService(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/services/%s", uuid), nil, nil); err != nil {
		return fmt.Errorf("deleting service %s: %w", uuid, err)
	}
	return nil
}
func (c *Client) StartService(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/services/%s/start", uuid), nil, nil); err != nil {
		return fmt.Errorf("starting service %s: %w", uuid, err)
	}
	return nil
}
func (c *Client) StopService(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/services/%s/stop", uuid), nil, nil); err != nil {
		return fmt.Errorf("stopping service %s: %w", uuid, err)
	}
	return nil
}

// RestartService restarts a service.
func (c *Client) RestartService(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/services/%s/restart", uuid), nil, nil); err != nil {
		return fmt.Errorf("restarting service %s: %w", uuid, err)
	}
	return nil
}

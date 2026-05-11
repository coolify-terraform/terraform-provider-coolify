package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Environment represents a Coolify environment within a project.
type Environment struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ProjectUUID string `json:"project_uuid"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// CreateEnvironmentInput is the input for creating an environment.
type CreateEnvironmentInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ListEnvironments returns all environments for the given project.
func (c *Client) ListEnvironments(ctx context.Context, projectUUID string) ([]Environment, error) {
	var r []Environment
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/environments", url.PathEscape(projectUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("listing environments for project %s: %w", projectUUID, err)
	}
	return r, nil
}

// GetEnvironment returns a single environment by name or UUID.
func (c *Client) GetEnvironment(ctx context.Context, projectUUID, nameOrUUID string) (*Environment, error) {
	var r Environment
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/%s", url.PathEscape(projectUUID), url.PathEscape(nameOrUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting environment %s in project %s: %w", nameOrUUID, projectUUID, err)
	}
	return &r, nil
}

// CreateEnvironment creates a new environment in the given project.
func (c *Client) CreateEnvironment(ctx context.Context, projectUUID string, input CreateEnvironmentInput) (*Environment, error) {
	var r Environment
	if err := c.doWithStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/environments", url.PathEscape(projectUUID)), input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating environment in project %s: %w", projectUUID, err)
	}
	return &r, nil
}

// DeleteEnvironment deletes an environment by name or UUID.
func (c *Client) DeleteEnvironment(ctx context.Context, projectUUID, nameOrUUID string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/projects/%s/environments/%s", url.PathEscape(projectUUID), url.PathEscape(nameOrUUID)), nil, nil); err != nil {
		return fmt.Errorf("deleting environment %s in project %s: %w", nameOrUUID, projectUUID, err)
	}
	return nil
}

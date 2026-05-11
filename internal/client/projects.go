package client

import (
	"context"
	"fmt"
	"net/http"
)

type Project struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type CreateProjectInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type UpdateProjectInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var r []Project
	if err := c.do(ctx, http.MethodGet, "/api/v1/projects", nil, &r); err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	return r, nil
}

func (c *Client) GetProject(ctx context.Context, uuid string) (*Project, error) {
	var r Project
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s", uuid), nil, &r); err != nil {
		return nil, fmt.Errorf("getting project %s: %w", uuid, err)
	}
	return &r, nil
}

func (c *Client) CreateProject(ctx context.Context, input CreateProjectInput) (*Project, error) {
	var r Project
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/projects", input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}
	return &r, nil
}

func (c *Client) UpdateProject(ctx context.Context, uuid string, input UpdateProjectInput) (*Project, error) {
	var r Project
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/projects/%s", uuid), input, &r); err != nil {
		return nil, fmt.Errorf("updating project %s: %w", uuid, err)
	}
	return &r, nil
}

func (c *Client) DeleteProject(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/projects/%s", uuid), nil, nil); err != nil {
		return fmt.Errorf("deleting project %s: %w", uuid, err)
	}
	return nil
}

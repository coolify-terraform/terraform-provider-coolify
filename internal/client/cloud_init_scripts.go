package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type CloudInitScript struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Script string `json:"script,omitempty"`
}

type CloudInitScriptInput struct {
	Name   string `json:"name,omitempty"`
	Script string `json:"script,omitempty"`
}

func (c *Client) ListCloudInitScripts(ctx context.Context) ([]CloudInitScript, error) {
	var r []CloudInitScript
	if err := c.do(ctx, http.MethodGet, "/api/v1/cloud-init-scripts", nil, &r); err != nil {
		return nil, fmt.Errorf("listing cloud-init scripts: %w", err)
	}
	return r, nil
}

func (c *Client) GetCloudInitScript(ctx context.Context, uuid string) (*CloudInitScript, error) {
	var r CloudInitScript
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/cloud-init-scripts/%s", url.PathEscape(uuid)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting cloud-init script %s: %w", uuid, err)
	}
	return &r, nil
}

func (c *Client) CreateCloudInitScript(ctx context.Context, input CloudInitScriptInput) (*CloudInitScript, error) {
	var r CloudInitScript
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/cloud-init-scripts", input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating cloud-init script: %w", err)
	}
	return &r, nil
}

func (c *Client) UpdateCloudInitScript(ctx context.Context, uuid string, input CloudInitScriptInput) (*CloudInitScript, error) {
	var r CloudInitScript
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/cloud-init-scripts/%s", url.PathEscape(uuid)), input, &r); err != nil {
		return nil, fmt.Errorf("updating cloud-init script %s: %w", uuid, err)
	}
	return &r, nil
}

func (c *Client) DeleteCloudInitScript(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/cloud-init-scripts/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting cloud-init script %s: %w", uuid, err)
	}
	return nil
}

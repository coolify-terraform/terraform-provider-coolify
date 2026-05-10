package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type Deployment struct {
	UUID       string `json:"uuid,omitempty"`
	ID         int    `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	ServerUUID string `json:"server_uuid,omitempty"`
}
type DeployByTagInput struct {
	ForceRebuild bool `json:"force_rebuild,omitempty"`
}

func (c *Client) ListDeployments(ctx context.Context) ([]Deployment, error) {
	var r []Deployment
	if err := c.do(ctx, http.MethodGet, "/api/v1/deployments", nil, &r); err != nil {
		return nil, fmt.Errorf("listing deployments: %w", err)
	}
	return r, nil
}
func (c *Client) GetDeployment(ctx context.Context, uuid string) (*Deployment, error) {
	var r Deployment
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/deployments/%s", uuid), nil, &r); err != nil {
		return nil, fmt.Errorf("getting deployment %s: %w", uuid, err)
	}
	return &r, nil
}
func (c *Client) ListApplicationDeployments(ctx context.Context, appUUID string) ([]Deployment, error) {
	var r []Deployment
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/deployments/applications/%s", appUUID), nil, &r); err != nil {
		return nil, fmt.Errorf("listing deployments for application %s: %w", appUUID, err)
	}
	return r, nil
}
func (c *Client) CancelDeployment(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/deployments/%s/cancel", uuid), nil, nil); err != nil {
		return fmt.Errorf("cancelling deployment %s: %w", uuid, err)
	}
	return nil
}

func (c *Client) DeployByTag(ctx context.Context, tag string, input DeployByTagInput) error {
	q := url.Values{}
	q.Set("tag", tag)
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/deploy?%s", q.Encode()), input, nil); err != nil {
		return fmt.Errorf("deploying by tag %s: %w", tag, err)
	}
	return nil
}

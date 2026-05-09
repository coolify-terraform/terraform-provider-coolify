package client

import (
	"context"
	"fmt"
	"net/http"
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
		return nil, err
	}
	return r, nil
}
func (c *Client) GetDeployment(ctx context.Context, uuid string) (*Deployment, error) {
	var r Deployment
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/deployments/%s", uuid), nil, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
func (c *Client) DeployByTag(ctx context.Context, tag string, input DeployByTagInput) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/deploy?tag=%s", tag), input, nil)
}

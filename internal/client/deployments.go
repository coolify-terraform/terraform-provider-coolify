package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Deployment struct {
	UUID       string `json:"deployment_uuid,omitempty"`
	ID         int    `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	ServerUUID string `json:"server_uuid,omitempty"`
}
type DeployByTagInput struct {
	ForceRebuild bool `json:"force_rebuild"`
}

func (c *Client) ListDeployments(ctx context.Context) ([]Deployment, error) {
	// Coolify bug: sortBy('id') without values() produces a JSON object
	// with non-sequential keys instead of an array when deployments have
	// gaps in their indices. Try array first, fall back to object.
	// See: https://github.com/coollabsio/coolify/issues/10077
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/api/v1/deployments", nil, &raw); err != nil {
		return nil, fmt.Errorf("listing deployments: %w", err)
	}
	var r []Deployment
	if err := json.Unmarshal(raw, &r); err == nil {
		return r, nil
	}
	var m map[string]Deployment
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("listing deployments: decoding response: %w", err)
	}
	r = make([]Deployment, 0, len(m))
	for _, d := range m {
		r = append(r, d)
	}
	return r, nil
}
func (c *Client) GetDeployment(ctx context.Context, uuid string) (*Deployment, error) {
	var r Deployment
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/deployments/%s", url.PathEscape(uuid)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting deployment %s: %w", uuid, err)
	}
	return &r, nil
}
func (c *Client) ListApplicationDeployments(ctx context.Context, appUUID string) ([]Deployment, error) {
	var r []Deployment
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/deployments/applications/%s", url.PathEscape(appUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("listing deployments for application %s: %w", appUUID, err)
	}
	return r, nil
}
func (c *Client) CancelDeployment(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/deployments/%s/cancel", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("cancelling deployment %s: %w", uuid, err)
	}
	return nil
}

// Deploy triggers a generic deploy (webhook-style).
func (c *Client) Deploy(ctx context.Context) error {
	if err := c.do(ctx, http.MethodGet, "/api/v1/deploy", nil, nil); err != nil {
		return fmt.Errorf("triggering deploy: %w", err)
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

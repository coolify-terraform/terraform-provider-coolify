package client

import (
	"context"
	"fmt"
	"net/http"
)

type Resource struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

func (c *Client) ListResources(ctx context.Context) ([]Resource, error) {
	var r []Resource
	if err := c.do(ctx, http.MethodGet, "/api/v1/resources", nil, &r); err != nil {
		return nil, fmt.Errorf("listing resources: %w", err)
	}
	return r, nil
}

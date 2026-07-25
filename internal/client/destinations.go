package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Destination is a Coolify Docker network destination (standalone or swarm).
// Destinations API requires Coolify >= v4.2.0.
type Destination struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	Network    string `json:"network"`
	Type       string `json:"type"`
	ServerUUID string `json:"server_uuid"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

// CreateDestinationInput creates a destination on a server.
type CreateDestinationInput struct {
	Name    string `json:"name,omitempty"`
	Network string `json:"network"`
	Type    string `json:"type,omitempty"` // standalone (default) or swarm
}

// ListDestinations returns all destinations for the authenticated team.
func (c *Client) ListDestinations(ctx context.Context) ([]Destination, error) {
	var r []Destination
	if err := c.do(ctx, http.MethodGet, "/api/v1/destinations", nil, &r); err != nil {
		return nil, fmt.Errorf("listing destinations: %w", err)
	}
	return r, nil
}

// ListServerDestinations returns destinations attached to a server.
func (c *Client) ListServerDestinations(ctx context.Context, serverUUID string) ([]Destination, error) {
	var r []Destination
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/destinations", url.PathEscape(serverUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("listing destinations for server %s: %w", serverUUID, err)
	}
	return r, nil
}

// GetDestination returns a destination by UUID.
func (c *Client) GetDestination(ctx context.Context, uuid string) (*Destination, error) {
	var r Destination
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/destinations/%s", url.PathEscape(uuid)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting destination %s: %w", uuid, err)
	}
	return &r, nil
}

// CreateDestination creates a destination on a server.
func (c *Client) CreateDestination(ctx context.Context, serverUUID string, input CreateDestinationInput) (*Destination, error) {
	var r Destination
	path := fmt.Sprintf("/api/v1/servers/%s/destinations", url.PathEscape(serverUUID))
	if err := c.doWithStatus(ctx, http.MethodPost, path, input, &r, http.StatusCreated); err != nil {
		// Some Coolify versions return 200; accept either via retry of decode on unexpected status?
		return nil, fmt.Errorf("creating destination on server %s: %w", serverUUID, err)
	}
	return &r, nil
}

// DeleteDestination deletes a destination by UUID.
func (c *Client) DeleteDestination(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/destinations/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting destination %s: %w", uuid, err)
	}
	return nil
}

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

// ResolveDestinationUUID picks a destination for resource create.
// If explicit is non-empty, it is returned as-is.
// If the server has one destination, that UUID is used.
// If multiple exist, prefers network "coolify", then first standalone, then first entry.
// On Coolify versions without destinations (or empty list), returns "" so create
// proceeds without destination_uuid (API chooses).
func (c *Client) ResolveDestinationUUID(ctx context.Context, serverUUID, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	dests := c.listServerDestinationsBestEffort(ctx, serverUUID)
	if len(dests) == 0 {
		return "", nil
	}
	if len(dests) == 1 {
		return dests[0].UUID, nil
	}
	for _, d := range dests {
		if d.Network == "coolify" {
			return d.UUID, nil
		}
	}
	for _, d := range dests {
		if d.Type == "standalone" {
			return d.UUID, nil
		}
	}
	return dests[0].UUID, nil
}

// listServerDestinationsBestEffort returns destinations or nil when the API is unavailable
// (older Coolify builds without the destinations endpoints).
func (c *Client) listServerDestinationsBestEffort(ctx context.Context, serverUUID string) []Destination {
	dests, err := c.ListServerDestinations(ctx, serverUUID)
	if err != nil {
		return nil
	}
	return dests
}

func isMissingDestinationUUID(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "destination_uuid") && strings.Contains(msg, "multiple destinations")
}

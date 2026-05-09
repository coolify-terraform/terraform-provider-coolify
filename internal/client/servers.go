package client

import (
	"context"
	"fmt"
	"net/http"
)

type Server struct {
	UUID           string `json:"uuid,omitempty"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	IP             string `json:"ip"`
	Port           int    `json:"port,omitempty"`
	User           string `json:"user,omitempty"`
	PrivateKeyUUID string `json:"private_key_uuid,omitempty"`
	IsBuildServer  bool   `json:"is_build_server"`
	IsReachable    bool   `json:"is_reachable"`
	IsUsable       bool   `json:"is_usable"`
}
type CreateServerInput struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	IP             string `json:"ip"`
	Port           int    `json:"port"`
	User           string `json:"user,omitempty"`
	PrivateKeyUUID string `json:"private_key_uuid"`
	IsBuildServer  bool   `json:"is_build_server,omitempty"`
}
type UpdateServerInput struct {
	Name           *string `json:"name,omitempty"`
	Description    *string `json:"description,omitempty"`
	IP             *string `json:"ip,omitempty"`
	Port           *int    `json:"port,omitempty"`
	User           *string `json:"user,omitempty"`
	PrivateKeyUUID *string `json:"private_key_uuid,omitempty"`
	IsBuildServer  *bool   `json:"is_build_server,omitempty"`
}

func (c *Client) ListServers(ctx context.Context) ([]Server, error) {
	var s []Server
	if err := c.do(ctx, http.MethodGet, "/api/v1/servers", nil, &s); err != nil {
		return nil, fmt.Errorf("listing servers: %w", err)
	}
	return s, nil
}
func (c *Client) GetServer(ctx context.Context, uuid string) (*Server, error) {
	var s Server
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s", uuid), nil, &s); err != nil {
		return nil, fmt.Errorf("getting server %s: %w", uuid, err)
	}
	return &s, nil
}
func (c *Client) CreateServer(ctx context.Context, input CreateServerInput) (*Server, error) {
	var s Server
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/servers", input, &s, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating server: %w", err)
	}
	return &s, nil
}
func (c *Client) UpdateServer(ctx context.Context, uuid string, input UpdateServerInput) (*Server, error) {
	var s Server
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/servers/%s", uuid), input, &s); err != nil {
		return nil, fmt.Errorf("updating server %s: %w", uuid, err)
	}
	return &s, nil
}
func (c *Client) DeleteServer(ctx context.Context, uuid string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/servers/%s", uuid), nil, nil)
}

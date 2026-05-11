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
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/servers/%s", uuid), nil, nil); err != nil {
		return fmt.Errorf("deleting server %s: %w", uuid, err)
	}
	return nil
}

// ServerValidation represents the result of a server connectivity check.
type ServerValidation struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
}

// ValidateServer triggers a connectivity check on the server.
// Coolify uses GET for this endpoint.
func (c *Client) ValidateServer(ctx context.Context, uuid string) (*ServerValidation, error) {
	var v ServerValidation
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/validate", uuid), nil, &v); err != nil {
		return nil, fmt.Errorf("validating server %s: %w", uuid, err)
	}
	return &v, nil
}

type ServerResource struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// ListServerResources returns all resources (apps, databases, services) deployed on a server.
func (c *Client) ListServerResources(ctx context.Context, uuid string) ([]ServerResource, error) {
	var r []ServerResource
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/resources", uuid), nil, &r); err != nil {
		return nil, fmt.Errorf("listing server resources %s: %w", uuid, err)
	}
	return r, nil
}

type ServerDomain struct {
	Domain string `json:"domain"`
	IP     string `json:"ip"`
}

type CreateHetznerServerInput struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	HetznerToken   string `json:"hetzner_token"`
	ServerType     string `json:"server_type"`
	Location       string `json:"location"`
	Image          string `json:"image"`
	SSHKeyName     string `json:"ssh_key_name,omitempty"`
	PrivateKeyUUID string `json:"private_key_uuid"`
	ProjectUUID    string `json:"project_uuid"`
}

func (c *Client) CreateHetznerServer(ctx context.Context, input CreateHetznerServerInput) (*Server, error) {
	var s Server
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/servers/hetzner", input, &s, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating hetzner server: %w", err)
	}
	return &s, nil
}

// ListServerDomains returns all domains configured on a server.
func (c *Client) ListServerDomains(ctx context.Context, uuid string) ([]ServerDomain, error) {
	var d []ServerDomain
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/domains", uuid), nil, &d); err != nil {
		return nil, fmt.Errorf("listing server domains %s: %w", uuid, err)
	}
	return d, nil
}

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// DockerRegistryLogin is one row from GET /servers/{uuid}/registries.
// Password is never returned.
type DockerRegistryLogin struct {
	Registry string `json:"registry"`
	LoggedIn bool   `json:"logged_in"`
	Source   string `json:"source,omitempty"`
	Username string `json:"username,omitempty"`
}

type dockerRegistryListResponse struct {
	Registries []DockerRegistryLogin `json:"registries"`
	Error      string                `json:"error"`
}

type dockerRegistryLoginInput struct {
	Registry string `json:"registry"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// NormalizeDockerRegistry matches Coolify DockerRegistryLogins::normalizeRegistry.
// Schemes and paths are stripped. Docker Hub host aliases become docker.io.
func NormalizeDockerRegistry(raw string) string {
	host := strings.ToLower(strings.TrimSpace(raw))
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	if i := strings.IndexByte(host, '/'); i >= 0 {
		host = host[:i]
	}
	switch host {
	case "index.docker.io", "registry-1.docker.io", "registry.hub.docker.com", "docker.io":
		return "docker.io"
	default:
		return host
	}
}

// ListServerDockerRegistries reads docker logins on a server.
// A non-empty error string with an empty login list means the server could
// not be read. Callers must keep Terraform state in that case.
func (c *Client) ListServerDockerRegistries(ctx context.Context, serverUUID string) ([]DockerRegistryLogin, error) {
	var out dockerRegistryListResponse
	path := fmt.Sprintf("/api/v1/servers/%s/registries", url.PathEscape(serverUUID))
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, fmt.Errorf("listing docker registries on server %s: %w", serverUUID, err)
	}
	if out.Error != "" {
		return nil, fmt.Errorf("listing docker registries on server %s: %s", serverUUID, out.Error)
	}
	if out.Registries == nil {
		return []DockerRegistryLogin{}, nil
	}
	return out.Registries, nil
}

// LoginServerDockerRegistry runs docker login on the server. Logging in
// again to the same registry replaces the saved credentials.
func (c *Client) LoginServerDockerRegistry(ctx context.Context, serverUUID, registry, username, password string) error {
	path := fmt.Sprintf("/api/v1/servers/%s/registries", url.PathEscape(serverUUID))
	body := dockerRegistryLoginInput{
		Registry: NormalizeDockerRegistry(registry),
		Username: username,
		Password: password,
	}
	if err := c.do(ctx, http.MethodPost, path, body, nil); err != nil {
		return fmt.Errorf("logging in to %s on server %s: %w", body.Registry, serverUUID, err)
	}
	return nil
}

// LogoutServerDockerRegistry runs docker logout for one registry.
func (c *Client) LogoutServerDockerRegistry(ctx context.Context, serverUUID, registry string) error {
	reg := NormalizeDockerRegistry(registry)
	path := fmt.Sprintf("/api/v1/servers/%s/registries/%s", url.PathEscape(serverUUID), url.PathEscape(reg))
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("logging out of %s on server %s: %w", reg, serverUUID, err)
	}
	return nil
}

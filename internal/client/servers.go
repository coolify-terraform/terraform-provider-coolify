package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ServerSettings holds the nested settings returned by the Coolify API
// inside the "settings" key of a Server object.
type ServerSettings struct {
	ConcurrentBuilds                     int    `json:"concurrent_builds"`
	DynamicTimeout                       int    `json:"dynamic_timeout"`
	DeploymentQueueLimit                 int    `json:"deployment_queue_limit"`
	ServerDiskUsageNotificationThreshold int    `json:"server_disk_usage_notification_threshold"`
	ServerDiskUsageCheckFrequency        string `json:"server_disk_usage_check_frequency"`
	WildcardDomain                       string `json:"wildcard_domain,omitempty"`
	IsCloudFlareTunnel                   bool   `json:"is_cloudflare_tunnel"`
	ServerTimezone                       string `json:"server_timezone,omitempty"`
	IsMetricsEnabled                     bool   `json:"is_metrics_enabled"`
	IsTerminalEnabled                    bool   `json:"is_terminal_enabled"`
	IsSentinelEnabled                    bool   `json:"is_sentinel_enabled"`
	SentinelToken                        string `json:"sentinel_token,omitempty"`
	SentinelCustomURL                    string `json:"sentinel_custom_url,omitempty"`
	SentinelMetricsHistoryDays           int    `json:"sentinel_metrics_history_days"`
	SentinelMetricsRefreshRateSeconds    int    `json:"sentinel_metrics_refresh_rate_seconds"`
	SentinelPushIntervalSeconds          int    `json:"sentinel_push_interval_seconds"`
	DockerCleanupFrequency               string `json:"docker_cleanup_frequency,omitempty"`
	DockerCleanupThreshold               int    `json:"docker_cleanup_threshold"`
	ForceDockerCleanup                   bool   `json:"force_docker_cleanup"`
	DeleteUnusedVolumes                  bool   `json:"delete_unused_volumes"`
	DeleteUnusedNetworks                 bool   `json:"delete_unused_networks"`
	GenerateExactLabels                  bool   `json:"generate_exact_labels"`
}

type Server struct {
	UUID           string          `json:"uuid,omitempty"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	IP             string          `json:"ip"`
	Port           int             `json:"port,omitempty"`
	User           string          `json:"user,omitempty"`
	PrivateKeyUUID string          `json:"private_key_uuid,omitempty"`
	IsBuildServer  bool            `json:"is_build_server"`
	IsReachable    bool            `json:"is_reachable"`
	IsUsable       bool            `json:"is_usable"`
	Settings       *ServerSettings `json:"settings,omitempty"`
}
type CreateServerInput struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	IP             string `json:"ip"`
	Port           int    `json:"port"`
	User           string `json:"user,omitempty"`
	PrivateKeyUUID string `json:"private_key_uuid"`
	IsBuildServer  *bool  `json:"is_build_server,omitempty"`
}
type UpdateServerInput struct {
	Name                                 *string `json:"name,omitempty"`
	Description                          *string `json:"description,omitempty"`
	IP                                   *string `json:"ip,omitempty"`
	Port                                 *int    `json:"port,omitempty"`
	User                                 *string `json:"user,omitempty"`
	PrivateKeyUUID                       *string `json:"private_key_uuid,omitempty"`
	IsBuildServer                        *bool   `json:"is_build_server,omitempty"`
	ConcurrentBuilds                     *int    `json:"concurrent_builds,omitempty"`
	DynamicTimeout                       *int    `json:"dynamic_timeout,omitempty"`
	DeploymentQueueLimit                 *int    `json:"deployment_queue_limit,omitempty"`
	ServerDiskUsageNotificationThreshold *int    `json:"server_disk_usage_notification_threshold,omitempty"`
	ServerDiskUsageCheckFrequency        *string `json:"server_disk_usage_check_frequency,omitempty"`
	WildcardDomain                       *string `json:"wildcard_domain,omitempty"`
	IsCloudFlareTunnel                   *bool   `json:"is_cloudflare_tunnel,omitempty"`
	ServerTimezone                       *string `json:"server_timezone,omitempty"`
	IsMetricsEnabled                     *bool   `json:"is_metrics_enabled,omitempty"`
	IsTerminalEnabled                    *bool   `json:"is_terminal_enabled,omitempty"`
	IsSentinelEnabled                    *bool   `json:"is_sentinel_enabled,omitempty"`
	SentinelMetricsHistoryDays           *int    `json:"sentinel_metrics_history_days,omitempty"`
	SentinelMetricsRefreshRateSeconds    *int    `json:"sentinel_metrics_refresh_rate_seconds,omitempty"`
	SentinelPushIntervalSeconds          *int    `json:"sentinel_push_interval_seconds,omitempty"`
	DockerCleanupFrequency               *string `json:"docker_cleanup_frequency,omitempty"`
	DockerCleanupThreshold               *int    `json:"docker_cleanup_threshold,omitempty"`
	ForceDockerCleanup                   *bool   `json:"force_docker_cleanup,omitempty"`
	DeleteUnusedVolumes                  *bool   `json:"delete_unused_volumes,omitempty"`
	DeleteUnusedNetworks                 *bool   `json:"delete_unused_networks,omitempty"`
	GenerateExactLabels                  *bool   `json:"generate_exact_labels,omitempty"`
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
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s", url.PathEscape(uuid)), nil, &s); err != nil {
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
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/servers/%s", url.PathEscape(uuid)), input, &s); err != nil {
		return nil, fmt.Errorf("updating server %s: %w", uuid, err)
	}
	return &s, nil
}
func (c *Client) DeleteServer(ctx context.Context, uuid string) error {
	path := fmt.Sprintf("/api/v1/servers/%s?force=true", url.PathEscape(uuid))
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
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
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/validate", url.PathEscape(uuid)), nil, &v); err != nil {
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
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/resources", url.PathEscape(uuid)), nil, &r); err != nil {
		return nil, fmt.Errorf("listing server resources %s: %w", uuid, err)
	}
	return r, nil
}

type ServerDomain struct {
	Domain string `json:"domain"`
	IP     string `json:"ip"`
}

type CreateHetznerServerInput struct {
	Name                   string `json:"name"`
	CloudProviderTokenUUID string `json:"cloud_provider_token_uuid"`
	ServerType             string `json:"server_type"`
	Location               string `json:"location"`
	Image                  string `json:"image"`
	PrivateKeyUUID         string `json:"private_key_uuid"`
	EnableIPv4             *bool  `json:"enable_ipv4,omitempty"`
	EnableIPv6             *bool  `json:"enable_ipv6,omitempty"`
	HetznerSSHKeyIDs       string `json:"hetzner_ssh_key_ids,omitempty"`
	CloudInitScript        string `json:"cloud_init_script,omitempty"`
	InstantValidate        *bool  `json:"instant_validate,omitempty"`
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
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/domains", url.PathEscape(uuid)), nil, &d); err != nil {
		return nil, fmt.Errorf("listing server domains %s: %w", uuid, err)
	}
	return d, nil
}

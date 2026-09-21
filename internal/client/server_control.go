package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type ServerProxy struct {
	RedirectEnabled     *bool  `json:"redirect_enabled,omitempty"`
	RedirectURL         string `json:"redirect_url,omitempty"`
	GenerateExactLabels *bool  `json:"generate_exact_labels,omitempty"`
	ProxyType           string `json:"proxy_type,omitempty"`
	Configuration       string `json:"configuration,omitempty"`
}

type ServerProxyUpdateInput struct {
	RedirectEnabled     *bool   `json:"redirect_enabled,omitempty"`
	RedirectURL         *string `json:"redirect_url,omitempty"`
	GenerateExactLabels *bool   `json:"generate_exact_labels,omitempty"`
	ProxyType           *string `json:"proxy_type,omitempty"`
}

type ServerProxyConfigInput struct {
	Configuration string `json:"configuration"`
}

func (c *Client) GetServerProxy(ctx context.Context, serverUUID string) (*ServerProxy, error) {
	var r ServerProxy
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/proxy", url.PathEscape(serverUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting server proxy %s: %w", serverUUID, err)
	}
	return &r, nil
}

func (c *Client) UpdateServerProxy(ctx context.Context, serverUUID string, input ServerProxyUpdateInput) (*ServerProxy, error) {
	var r ServerProxy
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/servers/%s/proxy", url.PathEscape(serverUUID)), input, &r); err != nil {
		return nil, fmt.Errorf("updating server proxy %s: %w", serverUUID, err)
	}
	return &r, nil
}

func (c *Client) PutServerProxyConfiguration(ctx context.Context, serverUUID, configuration string) error {
	input := ServerProxyConfigInput{Configuration: configuration}
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/api/v1/servers/%s/proxy/configuration", url.PathEscape(serverUUID)), input, nil); err != nil {
		return fmt.Errorf("saving server proxy configuration %s: %w", serverUUID, err)
	}
	return nil
}

type ServerLogDrains struct {
	IsNewRelicEnabled  *bool  `json:"is_logdrain_newrelic_enabled,omitempty"`
	NewRelicLicenseKey string `json:"logdrain_newrelic_license_key,omitempty"`
	NewRelicBaseURI    string `json:"logdrain_newrelic_base_uri,omitempty"`
	IsAxiomEnabled     *bool  `json:"is_logdrain_axiom_enabled,omitempty"`
	AxiomDatasetName   string `json:"logdrain_axiom_dataset_name,omitempty"`
	AxiomAPIKey        string `json:"logdrain_axiom_api_key,omitempty"`
	IsCustomEnabled    *bool  `json:"is_logdrain_custom_enabled,omitempty"`
	CustomConfig       string `json:"logdrain_custom_config,omitempty"`
	CustomConfigParser string `json:"logdrain_custom_config_parser,omitempty"`
}

func (c *Client) GetServerLogDrains(ctx context.Context, serverUUID string) (*ServerLogDrains, error) {
	var r ServerLogDrains
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/log-drains", url.PathEscape(serverUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting server log drains %s: %w", serverUUID, err)
	}
	return &r, nil
}

func (c *Client) UpdateServerLogDrains(ctx context.Context, serverUUID string, input ServerLogDrains) (*ServerLogDrains, error) {
	var r ServerLogDrains
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/servers/%s/log-drains", url.PathEscape(serverUUID)), input, &r); err != nil {
		return nil, fmt.Errorf("updating server log drains %s: %w", serverUUID, err)
	}
	return &r, nil
}

type ServerCloudflareTunnel struct {
	IsCloudflareTunnel bool `json:"is_cloudflare_tunnel"`
}

func (c *Client) GetServerCloudflareTunnel(ctx context.Context, serverUUID string) (*ServerCloudflareTunnel, error) {
	var r ServerCloudflareTunnel
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/cloudflare-tunnel", url.PathEscape(serverUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting server cloudflare tunnel %s: %w", serverUUID, err)
	}
	return &r, nil
}

func (c *Client) UpdateServerCloudflareTunnel(ctx context.Context, serverUUID string, enabled bool) (*ServerCloudflareTunnel, error) {
	var r ServerCloudflareTunnel
	input := ServerCloudflareTunnel{IsCloudflareTunnel: enabled}
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/servers/%s/cloudflare-tunnel", url.PathEscape(serverUUID)), input, &r); err != nil {
		return nil, fmt.Errorf("updating server cloudflare tunnel %s: %w", serverUUID, err)
	}
	return &r, nil
}

type ServerSentinel struct {
	IsSentinelEnabled                 *bool  `json:"is_sentinel_enabled,omitempty"`
	IsMetricsEnabled                  *bool  `json:"is_metrics_enabled,omitempty"`
	IsSentinelDebugEnabled            *bool  `json:"is_sentinel_debug_enabled,omitempty"`
	SentinelToken                     string `json:"sentinel_token,omitempty"`
	SentinelMetricsRefreshRateSeconds *int64 `json:"sentinel_metrics_refresh_rate_seconds,omitempty"`
	SentinelMetricsHistoryDays        *int64 `json:"sentinel_metrics_history_days,omitempty"`
	SentinelPushIntervalSeconds       *int64 `json:"sentinel_push_interval_seconds,omitempty"`
	SentinelCustomURL                 string `json:"sentinel_custom_url,omitempty"`
	// Coolify >= 4.4 tip (ServerSentinelController ALLOWED_FIELDS). Not in
	// v4.3.23 or v4.4-rc.1. is_traffic_analytics_enabled is fillable on
	// ServerSetting but is not on this allow list.
	TrafficTopN            *int64 `json:"traffic_topn,omitempty"`
	TrafficSampleThreshold *int64 `json:"traffic_sample_threshold,omitempty"`
	TrafficRetention1hDays *int64 `json:"traffic_retention_1h_days,omitempty"`
	TrafficRetention1dDays *int64 `json:"traffic_retention_1d_days,omitempty"`
	IsGeoIPEnabled         *bool  `json:"is_geoip_enabled,omitempty"`
	GeoIPRefreshDays       *int64 `json:"geoip_refresh_days,omitempty"`
	GeoIPMaxMindLicenseKey string `json:"geoip_maxmind_license_key,omitempty"`
}

func (c *Client) GetServerSentinel(ctx context.Context, serverUUID string) (*ServerSentinel, error) {
	var r ServerSentinel
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/sentinel", url.PathEscape(serverUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting server sentinel %s: %w", serverUUID, err)
	}
	return &r, nil
}

// IsSentinelEnabledNotAllowed reports a Coolify extra-key 422 for
// is_sentinel_enabled on PATCH /servers/{uuid}/sentinel. Tip after
// 2026-09-15 treats Sentinel as mandatory on regular servers, so that
// key is GET-only (readOnly in OpenAPI).
func IsSentinelEnabledNotAllowed(err error) bool {
	return APIMessageContains(err, "is_sentinel_enabled") && APIMessageContains(err, "not allowed")
}

func sentinelWriteEmpty(in ServerSentinel) bool {
	return in.IsSentinelEnabled == nil &&
		in.IsMetricsEnabled == nil &&
		in.IsSentinelDebugEnabled == nil &&
		in.SentinelToken == "" &&
		in.SentinelMetricsRefreshRateSeconds == nil &&
		in.SentinelMetricsHistoryDays == nil &&
		in.SentinelPushIntervalSeconds == nil &&
		in.SentinelCustomURL == "" &&
		in.TrafficTopN == nil &&
		in.TrafficSampleThreshold == nil &&
		in.TrafficRetention1hDays == nil &&
		in.TrafficRetention1dDays == nil &&
		in.IsGeoIPEnabled == nil &&
		in.GeoIPRefreshDays == nil &&
		in.GeoIPMaxMindLicenseKey == ""
}

func (s *ServerSentinel) clearTrafficSettings() {
	s.TrafficTopN = nil
	s.TrafficSampleThreshold = nil
	s.TrafficRetention1hDays = nil
	s.TrafficRetention1dDays = nil
	s.IsGeoIPEnabled = nil
	s.GeoIPRefreshDays = nil
	s.GeoIPMaxMindLicenseKey = ""
}

func (c *Client) UpdateServerSentinel(ctx context.Context, serverUUID string, input ServerSentinel) (*ServerSentinel, error) {
	if !c.SupportsSentinelTrafficSettings() {
		input.clearTrafficSettings()
	}
	path := fmt.Sprintf("/api/v1/servers/%s/sentinel", url.PathEscape(serverUUID))
	var r ServerSentinel
	if err := c.do(ctx, http.MethodPatch, path, input, &r); err != nil {
		return c.updateServerSentinelAfterError(ctx, serverUUID, path, input, err)
	}
	return &r, nil
}

func (c *Client) updateServerSentinelAfterError(ctx context.Context, serverUUID, path string, input ServerSentinel, err error) (*ServerSentinel, error) {
	if input.IsSentinelEnabled == nil || !IsSentinelEnabledNotAllowed(err) {
		return nil, fmt.Errorf("updating server sentinel %s: %w", serverUUID, err)
	}
	// Coolify extra-key 422s the enable flag. Retry without it so
	// metrics/token writes still land. An enable-only payload becomes a GET.
	stripped := input
	stripped.IsSentinelEnabled = nil
	if sentinelWriteEmpty(stripped) {
		got, getErr := c.GetServerSentinel(ctx, serverUUID)
		if getErr != nil {
			return nil, fmt.Errorf("updating server sentinel %s: %w", serverUUID, getErr)
		}
		return got, nil
	}
	var r ServerSentinel
	if retryErr := c.do(ctx, http.MethodPatch, path, stripped, &r); retryErr != nil {
		return nil, fmt.Errorf("updating server sentinel %s: %w", serverUUID, retryErr)
	}
	return &r, nil
}

type ServerDockerCleanup struct {
	DockerCleanupFrequency           string `json:"docker_cleanup_frequency,omitempty"`
	DockerCleanupThreshold           *int64 `json:"docker_cleanup_threshold,omitempty"`
	ForceDockerCleanup               *bool  `json:"force_docker_cleanup,omitempty"`
	DeleteUnusedVolumes              *bool  `json:"delete_unused_volumes,omitempty"`
	DeleteUnusedNetworks             *bool  `json:"delete_unused_networks,omitempty"`
	DisableApplicationImageRetention *bool  `json:"disable_application_image_retention,omitempty"`
}

func (c *Client) GetServerDockerCleanup(ctx context.Context, serverUUID string) (*ServerDockerCleanup, error) {
	var r ServerDockerCleanup
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/servers/%s/docker-cleanup", url.PathEscape(serverUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting server docker cleanup %s: %w", serverUUID, err)
	}
	return &r, nil
}

func (c *Client) UpdateServerDockerCleanup(ctx context.Context, serverUUID string, input ServerDockerCleanup) (*ServerDockerCleanup, error) {
	var r ServerDockerCleanup
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/servers/%s/docker-cleanup", url.PathEscape(serverUUID)), input, &r); err != nil {
		return nil, fmt.Errorf("updating server docker cleanup %s: %w", serverUUID, err)
	}
	return &r, nil
}

type ApplicationDestination struct {
	UUID      string `json:"uuid"`
	IsPrimary bool   `json:"is_primary,omitempty"`
}

func (c *Client) ListApplicationDestinations(ctx context.Context, appUUID string) ([]ApplicationDestination, error) {
	var r []ApplicationDestination
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s/destinations", url.PathEscape(appUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("listing destinations for application %s: %w", appUUID, err)
	}
	return r, nil
}

func (c *Client) AttachApplicationDestination(ctx context.Context, appUUID, destUUID string) error {
	body := map[string]string{"destination_uuid": destUUID}
	path := fmt.Sprintf("/api/v1/applications/%s/destinations", url.PathEscape(appUUID))
	if err := c.doWithStatus(ctx, http.MethodPost, path, body, nil, http.StatusCreated); err != nil {
		return fmt.Errorf("attaching destination %s to application %s: %w", destUUID, appUUID, err)
	}
	return nil
}

func (c *Client) DetachApplicationDestination(ctx context.Context, appUUID, destUUID string) error {
	path := fmt.Sprintf("/api/v1/applications/%s/destinations/%s", url.PathEscape(appUUID), url.PathEscape(destUUID))
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("detaching destination %s from application %s: %w", destUUID, appUUID, err)
	}
	return nil
}

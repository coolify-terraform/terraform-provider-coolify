package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// UpdateServiceApplicationInput is a PATCH body. Nil fields are omitted.
// is_permanent_redirect_enabled and traefik_certificate_resolver require
// Coolify >= v4.5.0. max_restart_count requires Coolify >= v4.3.19 and is
// rejected on v4.4-rc.1. is_force_https_enabled requires >= v4.3.8.
// noindex_domains requires >= v4.3.0.
type UpdateServiceApplicationInput struct {
	URL                        *string   `json:"url,omitempty"`
	NoindexDomains             *[]string `json:"noindex_domains,omitempty"`
	HumanName                  *string   `json:"human_name,omitempty"`
	Description                *string   `json:"description,omitempty"`
	Image                      *string   `json:"image,omitempty"`
	ExcludeFromStatus          *bool     `json:"exclude_from_status,omitempty"`
	IsLogDrainEnabled          *bool     `json:"is_log_drain_enabled,omitempty"`
	IsGzipEnabled              *bool     `json:"is_gzip_enabled,omitempty"`
	IsStripPrefixEnabled       *bool     `json:"is_stripprefix_enabled,omitempty"`
	IsForceHTTPSEnabled        *bool     `json:"is_force_https_enabled,omitempty"`
	IsPermanentRedirectEnabled *bool     `json:"is_permanent_redirect_enabled,omitempty"`
	TraefikCertificateResolver *string   `json:"traefik_certificate_resolver,omitempty"`
	MaxRestartCount            *int64    `json:"max_restart_count,omitempty"`
}

func (c *Client) ListServiceApplications(ctx context.Context, serviceUUID string) ([]ServiceApplication, error) {
	var out []ServiceApplication
	path := fmt.Sprintf("/api/v1/services/%s/applications", url.PathEscape(serviceUUID))
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, fmt.Errorf("listing service applications for %s: %w", serviceUUID, err)
	}
	return out, nil
}

func (c *Client) GetServiceApplication(ctx context.Context, serviceUUID, appUUID string) (*ServiceApplication, error) {
	var out ServiceApplication
	path := fmt.Sprintf("/api/v1/services/%s/applications/%s", url.PathEscape(serviceUUID), url.PathEscape(appUUID))
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, fmt.Errorf("getting service application %s/%s: %w", serviceUUID, appUUID, err)
	}
	return &out, nil
}

func (c *Client) UpdateServiceApplication(ctx context.Context, serviceUUID, appUUID string, input UpdateServiceApplicationInput, forceDomainOverride bool) (*ServiceApplication, error) {
	if !c.SupportsCoolify45() {
		input.IsPermanentRedirectEnabled = nil
		input.TraefikCertificateResolver = nil
	}
	if !c.SupportsApplicationSettingsV43() {
		input.NoindexDomains = nil
	}
	if !c.SupportsServiceApplicationForceHTTPS() {
		input.IsForceHTTPSEnabled = nil
	}
	if !c.SupportsServiceApplicationMaxRestart() {
		input.MaxRestartCount = nil
	}
	path := fmt.Sprintf("/api/v1/services/%s/applications/%s", url.PathEscape(serviceUUID), url.PathEscape(appUUID))
	if forceDomainOverride {
		path += "?force_domain_override=true"
	}
	var out ServiceApplication
	if err := c.do(ctx, http.MethodPatch, path, input, &out); err != nil {
		return nil, fmt.Errorf("updating service application %s/%s: %w", serviceUUID, appUUID, err)
	}
	return &out, nil
}

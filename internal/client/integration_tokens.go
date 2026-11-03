package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// IntegrationToken is the public Coolify integration token. The secret and
// provider metadata are never returned.
type IntegrationToken struct {
	UUID          string   `json:"uuid"`
	Name          string   `json:"name"`
	Provider      string   `json:"provider"`
	Capabilities  []string `json:"capabilities,omitempty"`
	AutomaticDNS  *bool    `json:"automatic_dns,omitempty"`
	ProxiedDNS    *bool    `json:"proxied_dns,omitempty"`
	DNSZonesCount *int64   `json:"dns_zones_count,omitempty"`
	ZonesSyncedAt *string  `json:"zones_synced_at,omitempty"`
	CreatedAt     string   `json:"created_at,omitempty"`
	UpdatedAt     string   `json:"updated_at,omitempty"`
}

// IntegrationTokenMetadata is secret-manager connection settings.
// Cloudflare rejects this object. Infisical requires base_url and client_id.
// Vault requires base_url.
type IntegrationTokenMetadata struct {
	BaseURL   string `json:"base_url,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Timeout   *int64 `json:"timeout,omitempty"`
}

// CreateIntegrationTokenInput is POST /security/integration-tokens.
// Coolify >= v4.6.0. Create validates the token with the upstream provider.
type CreateIntegrationTokenInput struct {
	Provider     string                    `json:"provider"`
	Name         string                    `json:"name"`
	Token        string                    `json:"token"`
	Capabilities []string                  `json:"capabilities,omitempty"`
	AutomaticDNS *bool                     `json:"automatic_dns,omitempty"`
	ProxiedDNS   *bool                     `json:"proxied_dns,omitempty"`
	Metadata     *IntegrationTokenMetadata `json:"metadata,omitempty"`
}

// UpdateIntegrationTokenInput is PATCH /security/integration-tokens/{uuid}.
// Omitted fields keep their stored value. Sending token, capabilities, or
// settings makes Coolify check the secret again.
// Metadata is a raw object so a cleared timeout can be JSON null. Coolify
// merges the object, then drops empty values, which removes that key.
type UpdateIntegrationTokenInput struct {
	Name         *string         `json:"name,omitempty"`
	Token        *string         `json:"token,omitempty"`
	Capabilities *[]string       `json:"capabilities,omitempty"`
	AutomaticDNS *bool           `json:"automatic_dns,omitempty"`
	ProxiedDNS   *bool           `json:"proxied_dns,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

func (c *Client) ListIntegrationTokens(ctx context.Context) ([]IntegrationToken, error) {
	var out []IntegrationToken
	if err := c.do(ctx, http.MethodGet, "/api/v1/security/integration-tokens", nil, &out); err != nil {
		return nil, fmt.Errorf("listing integration tokens: %w", err)
	}
	return out, nil
}

func (c *Client) GetIntegrationToken(ctx context.Context, uuid string) (*IntegrationToken, error) {
	var out IntegrationToken
	path := fmt.Sprintf("/api/v1/security/integration-tokens/%s", url.PathEscape(uuid))
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, fmt.Errorf("getting integration token %s: %w", uuid, err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("getting integration token %s: API returned empty resource", uuid)
	}
	return &out, nil
}

func (c *Client) CreateIntegrationToken(ctx context.Context, input CreateIntegrationTokenInput) (*IntegrationToken, error) {
	var created struct {
		UUID string `json:"uuid"`
	}
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/security/integration-tokens", input, &created, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating integration token: %w", err)
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("creating integration token: API returned empty UUID")
	}
	return &IntegrationToken{UUID: created.UUID, Name: input.Name, Provider: input.Provider}, nil
}

func (c *Client) UpdateIntegrationToken(ctx context.Context, uuid string, input UpdateIntegrationTokenInput) (*IntegrationToken, error) {
	var out IntegrationToken
	path := fmt.Sprintf("/api/v1/security/integration-tokens/%s", url.PathEscape(uuid))
	if err := c.do(ctx, http.MethodPatch, path, input, &out); err != nil {
		return nil, fmt.Errorf("updating integration token %s: %w", uuid, err)
	}
	return &out, nil
}

func (c *Client) DeleteIntegrationToken(ctx context.Context, uuid string) error {
	path := fmt.Sprintf("/api/v1/security/integration-tokens/%s", url.PathEscape(uuid))
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("deleting integration token %s: %w", uuid, err)
	}
	return nil
}

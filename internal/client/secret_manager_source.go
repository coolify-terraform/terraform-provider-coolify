package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// SecretManagerSource is GET /{resource_type}/{uuid}/secret-manager.
// AvailableTokens is intentionally absent: Coolify includes every team
// token with the secrets capability, and that list is not source state.
type SecretManagerSource struct {
	Linked               bool              `json:"linked"`
	IntegrationTokenUUID *string           `json:"integration_token_uuid"`
	Provider             *string           `json:"provider"`
	Settings             map[string]string `json:"settings"`
}

// SetSecretManagerSourceInput is PATCH /{resource_type}/{uuid}/secret-manager.
// Settings is always sent. An empty object stores null only when the token
// does not require settings. A nil map would send JSON null, so callers
// pass an empty map instead.
type SetSecretManagerSourceInput struct {
	IntegrationTokenUUID string            `json:"integration_token_uuid"`
	Settings             map[string]string `json:"settings"`
}

func secretManagerSourcePath(resourceType, uuid string) string {
	return fmt.Sprintf("/api/v1/%s/%s/secret-manager", url.PathEscape(resourceType), url.PathEscape(uuid))
}

func (c *Client) GetSecretManagerSource(ctx context.Context, resourceType, uuid string) (*SecretManagerSource, error) {
	var out SecretManagerSource
	path := secretManagerSourcePath(resourceType, uuid)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, fmt.Errorf("getting secret manager source for %s %s: %w", resourceType, uuid, err)
	}
	return &out, nil
}

func (c *Client) SetSecretManagerSource(ctx context.Context, resourceType, uuid string, input SetSecretManagerSourceInput) error {
	if input.Settings == nil {
		input.Settings = map[string]string{}
	}
	path := secretManagerSourcePath(resourceType, uuid)
	if err := c.do(ctx, http.MethodPatch, path, input, nil); err != nil {
		return fmt.Errorf("setting secret manager source for %s %s: %w", resourceType, uuid, err)
	}
	return nil
}

func (c *Client) DeleteSecretManagerSource(ctx context.Context, resourceType, uuid string) error {
	path := secretManagerSourcePath(resourceType, uuid)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("deleting secret manager source for %s %s: %w", resourceType, uuid, err)
	}
	return nil
}

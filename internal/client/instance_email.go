package client

import (
	"context"
	"fmt"
	"net/http"
)

// InstanceEmailSettings is the instance-wide SMTP/Resend config.
// GET/PATCH /settings/email. Requires Coolify >= v4.3.10 and a root-team token.
type InstanceEmailSettings struct {
	SMTPEnabled     bool    `json:"smtp_enabled"`
	SMTPFromAddress string  `json:"smtp_from_address,omitempty"`
	SMTPFromName    string  `json:"smtp_from_name,omitempty"`
	SMTPHost        string  `json:"smtp_host,omitempty"`
	SMTPPort        *int    `json:"smtp_port,omitempty"`
	SMTPEncryption  string  `json:"smtp_encryption,omitempty"`
	SMTPUsername    string  `json:"smtp_username,omitempty"`
	SMTPPassword    string  `json:"smtp_password,omitempty"`
	SMTPTimeout     *int    `json:"smtp_timeout,omitempty"`
	SMTPEhloDomain  *string `json:"smtp_ehlo_domain,omitempty"`
	ResendEnabled   bool    `json:"resend_enabled"`
	ResendAPIKey    string  `json:"resend_api_key,omitempty"`
}

// UpdateInstanceEmailInput is the PATCH body for /settings/email.
type UpdateInstanceEmailInput struct {
	SMTPEnabled     *bool   `json:"smtp_enabled,omitempty"`
	SMTPFromAddress *string `json:"smtp_from_address,omitempty"`
	SMTPFromName    *string `json:"smtp_from_name,omitempty"`
	SMTPHost        *string `json:"smtp_host,omitempty"`
	SMTPPort        *int    `json:"smtp_port,omitempty"`
	SMTPEncryption  *string `json:"smtp_encryption,omitempty"`
	SMTPUsername    *string `json:"smtp_username,omitempty"`
	SMTPPassword    *string `json:"smtp_password,omitempty"`
	SMTPTimeout     *int    `json:"smtp_timeout,omitempty"`
	SMTPEhloDomain  *string `json:"smtp_ehlo_domain,omitempty"`
	ResendEnabled   *bool   `json:"resend_enabled,omitempty"`
	ResendAPIKey    *string `json:"resend_api_key,omitempty"`
}

// InstanceEmailUpdateJSONTags returns JSON keys on UpdateInstanceEmailInput.
func InstanceEmailUpdateJSONTags() map[string]struct{} {
	return jsonTagsFromValue(UpdateInstanceEmailInput{})
}

// GetInstanceEmailSettings returns instance-wide SMTP/Resend settings.
// Coolify >= v4.3.10. Requires a root-team API token.
func (c *Client) GetInstanceEmailSettings(ctx context.Context) (*InstanceEmailSettings, error) {
	var r InstanceEmailSettings
	if err := c.do(ctx, http.MethodGet, "/api/v1/settings/email", nil, &r); err != nil {
		return nil, fmt.Errorf("getting instance email settings: %w", err)
	}
	return &r, nil
}

// UpdateInstanceEmailSettings patches instance-wide SMTP/Resend settings.
// Coolify >= v4.3.10. Requires a root-team API token with write:sensitive.
func (c *Client) UpdateInstanceEmailSettings(ctx context.Context, input UpdateInstanceEmailInput) (*InstanceEmailSettings, error) {
	var r InstanceEmailSettings
	if err := c.do(ctx, http.MethodPatch, "/api/v1/settings/email", input, &r); err != nil {
		return nil, fmt.Errorf("updating instance email settings: %w", err)
	}
	return &r, nil
}

// SupportsInstanceEmailSettings reports whether GET/PATCH /settings/email exists.
// The route landed in Coolify v4.3.10. Empty CoolifyVersion reports true.
func (c *Client) SupportsInstanceEmailSettings() bool {
	if c == nil || c.CoolifyVersion == "" {
		return true
	}
	return IsVersionAtLeast(c.CoolifyVersion, minSMTPEhloDomainVersion)
}

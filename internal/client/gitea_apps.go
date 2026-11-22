package client

import (
	"context"
	"fmt"
)

// GiteaApp is one row of GET /gitea-apps.
// Coolify can return client_secret, access_token, refresh_token, and
// webhook_secret when the token may read sensitive data. Those fields
// are not decoded.
type GiteaApp struct {
	ID           int64   `json:"id"`
	UUID         string  `json:"uuid"`
	Name         string  `json:"name"`
	Organization *string `json:"organization"`
	APIURL       string  `json:"api_url"`
	HTMLURL      string  `json:"html_url"`
	CustomUser   string  `json:"custom_user"`
	CustomPort   *int64  `json:"custom_port"`
	ClientID     *string `json:"client_id"`
	RedirectURI  *string `json:"redirect_uri"`
	IsSystemWide bool    `json:"is_system_wide"`
	IsPublic     bool    `json:"is_public"`
	IsConnected  bool    `json:"is_connected"`
	TeamID       *int64  `json:"team_id"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// ListGiteaApps calls GET /api/v1/gitea-apps.
// The list is the token's team sources plus system-wide sources.
func (c *Client) ListGiteaApps(ctx context.Context) ([]GiteaApp, error) {
	if err := c.requireCoolify47("listing gitea apps"); err != nil {
		return nil, err
	}
	var apps []GiteaApp
	if err := c.doCachedList(ctx, "/api/v1/gitea-apps", &apps); err != nil {
		return nil, fmt.Errorf("listing gitea apps: %w", err)
	}
	if apps == nil {
		apps = []GiteaApp{}
	}
	for i := range apps {
		if apps[i].UUID == "" {
			return nil, fmt.Errorf("listing gitea apps: app at index %d has an empty uuid", i)
		}
	}
	return apps, nil
}

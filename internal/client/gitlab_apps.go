package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// GitLabApp is a Coolify GitLab App source (Coolify >= v4.3.0).
type GitLabApp struct {
	ID           int64  `json:"id"`
	UUID         string `json:"uuid,omitempty"`
	Name         string `json:"name"`
	HTMLURL      string `json:"html_url,omitempty"`
	APIURL       string `json:"api_url,omitempty"`
	CustomUser   string `json:"custom_user,omitempty"`
	CustomPort   *int64 `json:"custom_port,omitempty"`
	GroupName    string `json:"group_name,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	WebhookToken string `json:"webhook_token,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	IsSystemWide bool   `json:"is_system_wide,omitempty"`
}

type CreateGitLabAppInput struct {
	Name         string `json:"name"`
	HTMLURL      string `json:"html_url"`
	APIURL       string `json:"api_url,omitempty"`
	CustomUser   string `json:"custom_user,omitempty"`
	CustomPort   *int64 `json:"custom_port,omitempty"`
	GroupName    string `json:"group_name,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	WebhookToken string `json:"webhook_token,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	IsSystemWide *bool  `json:"is_system_wide,omitempty"`
}

type UpdateGitLabAppInput struct {
	Name         *string `json:"name,omitempty"`
	HTMLURL      *string `json:"html_url,omitempty"`
	APIURL       *string `json:"api_url,omitempty"`
	CustomUser   *string `json:"custom_user,omitempty"`
	CustomPort   *int64  `json:"custom_port,omitempty"`
	GroupName    *string `json:"group_name,omitempty"`
	ClientID     *string `json:"client_id,omitempty"`
	ClientSecret *string `json:"client_secret,omitempty"`
	WebhookToken *string `json:"webhook_token,omitempty"`
	RedirectURI  *string `json:"redirect_uri,omitempty"`
	IsSystemWide *bool   `json:"is_system_wide,omitempty"`
}

func (c *Client) ListGitLabApps(ctx context.Context) ([]GitLabApp, error) {
	var r []GitLabApp
	if err := c.doCachedList(ctx, "/api/v1/gitlab-apps", &r); err != nil {
		return nil, fmt.Errorf("listing gitlab apps: %w", err)
	}
	return r, nil
}

func (c *Client) GetGitLabApp(ctx context.Context, id int64) (*GitLabApp, error) {
	apps, err := c.ListGitLabApps(ctx)
	if err != nil {
		return nil, err
	}
	for i := range apps {
		if apps[i].ID == id || apps[i].UUID == strconv.FormatInt(id, 10) {
			return &apps[i], nil
		}
	}
	return nil, &NotFoundError{Message: fmt.Sprintf("gitlab app %d not found", id)}
}

func (c *Client) GetGitLabAppByUUID(ctx context.Context, uuid string) (*GitLabApp, error) {
	apps, err := c.ListGitLabApps(ctx)
	if err != nil {
		return nil, err
	}
	for i := range apps {
		if apps[i].UUID == uuid || strconv.FormatInt(apps[i].ID, 10) == uuid {
			return &apps[i], nil
		}
	}
	return nil, &NotFoundError{Message: fmt.Sprintf("gitlab app %s not found", uuid)}
}

func (c *Client) CreateGitLabApp(ctx context.Context, input CreateGitLabAppInput) (*GitLabApp, error) {
	var raw json.RawMessage
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/gitlab-apps", input, &raw, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating gitlab app: %w", err)
	}
	c.listCache.invalidate("/api/v1/gitlab-apps")
	app, err := decodeGitLabApp(raw)
	if err != nil {
		return nil, fmt.Errorf("decoding gitlab app create response: %w", err)
	}
	if app.ID == 0 {
		found, getErr := c.resolveGitLabApp(ctx, app.UUID, input.Name)
		if getErr != nil {
			return nil, fmt.Errorf("resolving gitlab app after create: %w", getErr)
		}
		return found, nil
	}
	return app, nil
}

func (c *Client) resolveGitLabApp(ctx context.Context, uuid, name string) (*GitLabApp, error) {
	apps, err := c.ListGitLabApps(ctx)
	if err != nil {
		return nil, err
	}
	for i := range apps {
		if uuid != "" && apps[i].UUID == uuid && apps[i].ID != 0 {
			return &apps[i], nil
		}
		if name != "" && apps[i].Name == name && apps[i].ID != 0 {
			return &apps[i], nil
		}
	}
	return nil, fmt.Errorf("gitlab app %q not found after create", name)
}

func (c *Client) UpdateGitLabApp(ctx context.Context, id int64, input UpdateGitLabAppInput) (*GitLabApp, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("/api/v1/gitlab-apps/%s", url.PathEscape(strconv.FormatInt(id, 10)))
	if err := c.do(ctx, http.MethodPatch, path, input, &raw); err != nil {
		return nil, fmt.Errorf("updating gitlab app %d: %w", id, err)
	}
	c.listCache.invalidate("/api/v1/gitlab-apps")
	app, err := decodeGitLabApp(raw)
	if err != nil {
		return nil, fmt.Errorf("decoding gitlab app %d update response: %w", id, err)
	}
	return app, nil
}

type gitLabAppEnvelope struct {
	Data *GitLabApp `json:"data"`
}

func decodeGitLabApp(raw json.RawMessage) (*GitLabApp, error) {
	var wrapped gitLabAppEnvelope
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Data != nil && wrapped.Data.ID != 0 {
		return wrapped.Data, nil
	}
	var app GitLabApp
	if err := json.Unmarshal(raw, &app); err != nil {
		return nil, err
	}
	if app.ID == 0 && app.UUID == "" {
		return nil, fmt.Errorf("gitlab app response missing id")
	}
	return &app, nil
}

func (c *Client) DeleteGitLabApp(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/api/v1/gitlab-apps/%s", url.PathEscape(strconv.FormatInt(id, 10)))
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("deleting gitlab app %d: %w", id, err)
	}
	c.listCache.invalidate("/api/v1/gitlab-apps")
	return nil
}

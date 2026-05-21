package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type GitHubApp struct {
	ID               int64  `json:"id"`
	UUID             string `json:"uuid,omitempty"`
	Name             string `json:"name"`
	OrganizationName string `json:"organization,omitempty"`
	APIURL           string `json:"api_url,omitempty"`
	HTMLURL          string `json:"html_url,omitempty"`
	AppID            int64  `json:"app_id,omitempty"`
	InstallationID   int64  `json:"installation_id,omitempty"`
	ClientID         string `json:"client_id,omitempty"`
	WebhookSecret    string `json:"webhook_secret,omitempty"`
}

type CreateGitHubAppIntegrationInput struct {
	Name             string `json:"name"`
	OrganizationName string `json:"organization,omitempty"`
	APIURL           string `json:"api_url"`
	HTMLURL          string `json:"html_url"`
	AppID            int64  `json:"app_id"`
	InstallationID   int64  `json:"installation_id"`
	ClientID         string `json:"client_id"`
	ClientSecret     string `json:"client_secret"`
	WebhookSecret    string `json:"webhook_secret,omitempty"`
	PrivateKeyUUID   string `json:"private_key_uuid"`
}

type UpdateGitHubAppIntegrationInput struct {
	Name             *string `json:"name,omitempty"`
	OrganizationName *string `json:"organization,omitempty"`
	AppID            *int64  `json:"app_id,omitempty"`
	InstallationID   *int64  `json:"installation_id,omitempty"`
	ClientID         *string `json:"client_id,omitempty"`
	ClientSecret     *string `json:"client_secret,omitempty"`
	WebhookSecret    *string `json:"webhook_secret,omitempty"`
	PrivateKeyUUID   *string `json:"private_key_uuid,omitempty"`
}

type GitHubRepository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
}

type GitHubBranch struct {
	Name string `json:"name"`
}

type gitHubAppEnvelope struct {
	Message string     `json:"message,omitempty"`
	Data    *GitHubApp `json:"data,omitempty"`
}

type gitHubRepositoriesEnvelope struct {
	Repositories []GitHubRepository `json:"repositories"`
}

type gitHubBranchesEnvelope struct {
	Branches []GitHubBranch `json:"branches"`
}

func (c *Client) ListGitHubApps(ctx context.Context) ([]GitHubApp, error) {
	var apps []GitHubApp
	if err := c.doCachedList(ctx, "/api/v1/github-apps", &apps); err != nil {
		return nil, fmt.Errorf("listing github apps: %w", err)
	}
	for i := range apps {
		if err := validateGitHubAppResponse(&apps[i]); err != nil {
			return nil, fmt.Errorf("listing github apps: invalid app at index %d: %w", i, err)
		}
	}
	return apps, nil
}

func (c *Client) GetGitHubApp(ctx context.Context, id int64) (*GitHubApp, error) {
	var apps []GitHubApp
	if err := c.doCachedList(ctx, "/api/v1/github-apps", &apps); err != nil {
		return nil, fmt.Errorf("getting github app %d: %w", id, err)
	}
	for i := range apps {
		if apps[i].ID != id {
			continue
		}
		if err := validateGitHubAppResponse(&apps[i]); err != nil {
			return nil, fmt.Errorf("getting github app %d: invalid matched app: %w", id, err)
		}
		return &apps[i], nil
	}
	return nil, &NotFoundError{Message: fmt.Sprintf("github app %d not found", id)}
}

func (c *Client) CreateGitHubApp(ctx context.Context, input CreateGitHubAppIntegrationInput) (*GitHubApp, error) {
	var app GitHubApp
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/github-apps", input, &app, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating github app: %w", err)
	}
	c.listCache.invalidate("/api/v1/github-apps")
	if err := validateGitHubAppResponse(&app); err != nil {
		return nil, fmt.Errorf("creating github app: %w", err)
	}
	return &app, nil
}

func (c *Client) UpdateGitHubApp(ctx context.Context, id int64, input UpdateGitHubAppIntegrationInput) (*GitHubApp, error) {
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/github-apps/%d", id), input, &raw); err != nil {
		return nil, fmt.Errorf("updating github app %d: %w", id, err)
	}
	c.listCache.invalidate("/api/v1/github-apps")
	app, err := decodeGitHubApp(raw)
	if err != nil {
		return nil, fmt.Errorf("decoding github app %d update response: %w", id, err)
	}
	return app, nil
}

func (c *Client) DeleteGitHubApp(ctx context.Context, id int64) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/github-apps/%d", id), nil, nil); err != nil {
		return fmt.Errorf("deleting github app %d: %w", id, err)
	}
	c.listCache.invalidate("/api/v1/github-apps")
	return nil
}

func (c *Client) ListGitHubAppRepositories(ctx context.Context, id int64) ([]GitHubRepository, error) {
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/github-apps/%d/repositories", id), nil, &raw); err != nil {
		return nil, fmt.Errorf("listing github app %d repositories: %w", id, err)
	}
	repos, err := decodeGitHubRepositories(raw)
	if err != nil {
		return nil, fmt.Errorf("decoding github app %d repositories response: %w", id, err)
	}
	return repos, nil
}

func (c *Client) ListGitHubAppBranches(ctx context.Context, id int64, owner, repo string) ([]GitHubBranch, error) {
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/github-apps/%d/repositories/%s/%s/branches", id, url.PathEscape(owner), url.PathEscape(repo)), nil, &raw); err != nil {
		return nil, fmt.Errorf("listing github app %d branches for %s/%s: %w", id, owner, repo, err)
	}
	branches, err := decodeGitHubBranches(raw)
	if err != nil {
		return nil, fmt.Errorf("decoding github app %d branches response for %s/%s: %w", id, owner, repo, err)
	}
	return branches, nil
}

func decodeGitHubApp(raw json.RawMessage) (*GitHubApp, error) {
	var wrapped gitHubAppEnvelope
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Data != nil {
		if err := validateGitHubAppResponse(wrapped.Data); err != nil {
			return nil, err
		}
		return wrapped.Data, nil
	}

	var app GitHubApp
	if err := json.Unmarshal(raw, &app); err != nil {
		return nil, err
	}
	if err := validateGitHubAppResponse(&app); err != nil {
		return nil, err
	}
	return &app, nil
}

func validateGitHubAppResponse(app *GitHubApp) error {
	switch {
	case app == nil:
		return errors.New("github app response missing data")
	case app.ID == 0:
		return errors.New("github app response missing id")
	case app.Name == "":
		return errors.New("github app response missing name")
	default:
		return nil
	}
}

func decodeGitHubRepositories(raw json.RawMessage) ([]GitHubRepository, error) {
	var repos []GitHubRepository
	if err := json.Unmarshal(raw, &repos); err == nil {
		return repos, nil
	}

	var wrapped gitHubRepositoriesEnvelope
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Repositories, nil
}

func decodeGitHubBranches(raw json.RawMessage) ([]GitHubBranch, error) {
	var branches []GitHubBranch
	if err := json.Unmarshal(raw, &branches); err == nil {
		return branches, nil
	}

	var wrapped gitHubBranchesEnvelope
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Branches, nil
}

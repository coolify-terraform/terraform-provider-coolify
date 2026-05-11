package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type GitHubApp struct {
	ID               int64  `json:"id"`
	UUID             string `json:"uuid,omitempty"`
	Name             string `json:"name"`
	OrganizationName string `json:"organization,omitempty"`
	AppID            int64  `json:"app_id,omitempty"`
	InstallationID   int64  `json:"installation_id,omitempty"`
	ClientID         string `json:"client_id,omitempty"`
	WebhookSecret    string `json:"webhook_secret,omitempty"`
}

type CreateGitHubAppIntegrationInput struct {
	Name             string `json:"name"`
	OrganizationName string `json:"organization,omitempty"`
	AppID            int64  `json:"app_id"`
	InstallationID   int64  `json:"installation_id"`
	ClientID         string `json:"client_id"`
	ClientSecret     string `json:"client_secret"`
	WebhookSecret    string `json:"webhook_secret,omitempty"`
	PrivateKey       string `json:"private_key"`
}

type UpdateGitHubAppIntegrationInput struct {
	Name          *string `json:"name,omitempty"`
	WebhookSecret *string `json:"webhook_secret,omitempty"`
}

type GitHubRepository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
}

type GitHubBranch struct {
	Name string `json:"name"`
}

func (c *Client) ListGitHubApps(ctx context.Context) ([]GitHubApp, error) {
	var r []GitHubApp
	if err := c.do(ctx, http.MethodGet, "/api/v1/github-apps", nil, &r); err != nil {
		return nil, fmt.Errorf("listing github apps: %w", err)
	}
	return r, nil
}

func (c *Client) GetGitHubApp(ctx context.Context, id int64) (*GitHubApp, error) {
	apps, err := c.ListGitHubApps(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting github app %d: %w", id, err)
	}
	for i := range apps {
		if apps[i].ID == id {
			return &apps[i], nil
		}
	}
	return nil, &NotFoundError{Message: fmt.Sprintf("github app %d not found", id)}
}

func (c *Client) CreateGitHubApp(ctx context.Context, input CreateGitHubAppIntegrationInput) (*GitHubApp, error) {
	var r GitHubApp
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/github-apps", input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating github app: %w", err)
	}
	return &r, nil
}

func (c *Client) UpdateGitHubApp(ctx context.Context, id int64, input UpdateGitHubAppIntegrationInput) (*GitHubApp, error) {
	var r GitHubApp
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/github-apps/%d", id), input, &r); err != nil {
		return nil, fmt.Errorf("updating github app %d: %w", id, err)
	}
	return &r, nil
}

func (c *Client) DeleteGitHubApp(ctx context.Context, id int64) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/github-apps/%d", id), nil, nil); err != nil {
		return fmt.Errorf("deleting github app %d: %w", id, err)
	}
	return nil
}

func (c *Client) ListGitHubAppRepositories(ctx context.Context, id int64) ([]GitHubRepository, error) {
	var r []GitHubRepository
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/github-apps/%d/repositories", id), nil, &r); err != nil {
		return nil, fmt.Errorf("listing github app %d repositories: %w", id, err)
	}
	return r, nil
}

func (c *Client) ListGitHubAppBranches(ctx context.Context, id int64, owner, repo string) ([]GitHubBranch, error) {
	var r []GitHubBranch
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/github-apps/%d/repositories/%s/%s/branches", id, url.PathEscape(owner), url.PathEscape(repo)), nil, &r); err != nil {
		return nil, fmt.Errorf("listing github app %d branches for %s/%s: %w", id, owner, repo, err)
	}
	return r, nil
}

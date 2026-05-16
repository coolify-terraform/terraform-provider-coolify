package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type EnvironmentVariable struct {
	UUID      string `json:"uuid,omitempty"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsPreview bool   `json:"is_preview"`
	IsBuild   bool   `json:"is_buildtime"`
}

type applicationEnvVarInput struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsPreview bool   `json:"is_preview"`
	IsBuild   bool   `json:"is_buildtime"`
}

type envVarInput struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsPreview bool   `json:"is_preview"`
}

type CreateEnvVarResponse struct {
	UUID string `json:"uuid"`
}

func (c *Client) CreateApplicationEnvVar(ctx context.Context, appUUID string, ev EnvironmentVariable) (*CreateEnvVarResponse, error) {
	var r CreateEnvVarResponse
	input := applicationEnvVarInput{Key: ev.Key, Value: ev.Value, IsPreview: ev.IsPreview, IsBuild: ev.IsBuild}
	if err := c.doWithStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/applications/%s/envs", url.PathEscape(appUUID)), input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating application env var %s: %w", appUUID, err)
	}
	return &r, nil
}
func (c *Client) ListApplicationEnvVars(ctx context.Context, appUUID string) ([]EnvironmentVariable, error) {
	var v []EnvironmentVariable
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s/envs", url.PathEscape(appUUID)), nil, &v); err != nil {
		return nil, fmt.Errorf("listing application env vars %s: %w", appUUID, err)
	}
	return v, nil
}
func (c *Client) UpdateApplicationEnvVar(ctx context.Context, appUUID string, ev EnvironmentVariable) error {
	input := applicationEnvVarInput{Key: ev.Key, Value: ev.Value, IsPreview: ev.IsPreview, IsBuild: ev.IsBuild}
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/applications/%s/envs", url.PathEscape(appUUID)), input, nil); err != nil {
		return fmt.Errorf("updating application env var %s: %w", appUUID, err)
	}
	return nil
}
func (c *Client) DeleteApplicationEnvVar(ctx context.Context, appUUID string, envUUID string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/applications/%s/envs/%s", url.PathEscape(appUUID), url.PathEscape(envUUID)), nil, nil); err != nil {
		return fmt.Errorf("deleting application env var %s/%s: %w", appUUID, envUUID, err)
	}
	return nil
}
func (c *Client) CreateServiceEnvVar(ctx context.Context, svcUUID string, ev EnvironmentVariable) (*CreateEnvVarResponse, error) {
	var r CreateEnvVarResponse
	input := envVarInput{Key: ev.Key, Value: ev.Value, IsPreview: ev.IsPreview}
	if err := c.doWithStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/services/%s/envs", url.PathEscape(svcUUID)), input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating service env var %s: %w", svcUUID, err)
	}
	return &r, nil
}
func (c *Client) ListServiceEnvVars(ctx context.Context, svcUUID string) ([]EnvironmentVariable, error) {
	var v []EnvironmentVariable
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/services/%s/envs", url.PathEscape(svcUUID)), nil, &v); err != nil {
		return nil, fmt.Errorf("listing service env vars %s: %w", svcUUID, err)
	}
	return v, nil
}
func (c *Client) UpdateServiceEnvVar(ctx context.Context, svcUUID string, ev EnvironmentVariable) error {
	input := envVarInput{Key: ev.Key, Value: ev.Value, IsPreview: ev.IsPreview}
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/services/%s/envs", url.PathEscape(svcUUID)), input, nil); err != nil {
		return fmt.Errorf("updating service env var %s: %w", svcUUID, err)
	}
	return nil
}
func (c *Client) DeleteServiceEnvVar(ctx context.Context, svcUUID string, envUUID string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/services/%s/envs/%s", url.PathEscape(svcUUID), url.PathEscape(envUUID)), nil, nil); err != nil {
		return fmt.Errorf("deleting service env var %s/%s: %w", svcUUID, envUUID, err)
	}
	return nil
}
func (c *Client) CreateDatabaseEnvVar(ctx context.Context, dbUUID string, ev EnvironmentVariable) (*CreateEnvVarResponse, error) {
	var r CreateEnvVarResponse
	input := envVarInput{Key: ev.Key, Value: ev.Value, IsPreview: ev.IsPreview}
	if err := c.doWithStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/databases/%s/envs", url.PathEscape(dbUUID)), input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating database env var %s: %w", dbUUID, err)
	}
	return &r, nil
}
func (c *Client) ListDatabaseEnvVars(ctx context.Context, dbUUID string) ([]EnvironmentVariable, error) {
	var v []EnvironmentVariable
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/envs", url.PathEscape(dbUUID)), nil, &v); err != nil {
		return nil, fmt.Errorf("listing database env vars %s: %w", dbUUID, err)
	}
	return v, nil
}
func (c *Client) UpdateDatabaseEnvVar(ctx context.Context, dbUUID string, ev EnvironmentVariable) error {
	input := envVarInput{Key: ev.Key, Value: ev.Value, IsPreview: ev.IsPreview}
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/databases/%s/envs", url.PathEscape(dbUUID)), input, nil); err != nil {
		return fmt.Errorf("updating database env var %s: %w", dbUUID, err)
	}
	return nil
}
func (c *Client) DeleteDatabaseEnvVar(ctx context.Context, dbUUID string, envUUID string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/databases/%s/envs/%s", url.PathEscape(dbUUID), url.PathEscape(envUUID)), nil, nil); err != nil {
		return fmt.Errorf("deleting database env var %s/%s: %w", dbUUID, envUUID, err)
	}
	return nil
}

// --- Bulk environment variable types ---

// BulkEnvVarInput is the request payload for bulk environment variable updates.
type BulkEnvVarInput struct {
	Variables []EnvVarEntry `json:"data"`
}

// EnvVarEntry represents a single environment variable in a bulk update.
type EnvVarEntry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsPreview bool   `json:"is_preview"`
}

// BulkUpdateAppEnvVars performs a bulk update of environment variables on an application.
func (c *Client) BulkUpdateAppEnvVars(ctx context.Context, appUUID string, input BulkEnvVarInput) error {
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/applications/%s/envs/bulk", url.PathEscape(appUUID)), input, nil); err != nil {
		return fmt.Errorf("bulk updating application env vars %s: %w", appUUID, err)
	}
	return nil
}

// BulkUpdateDatabaseEnvVars performs a bulk update of environment variables on a database.
func (c *Client) BulkUpdateDatabaseEnvVars(ctx context.Context, dbUUID string, input BulkEnvVarInput) error {
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/databases/%s/envs/bulk", url.PathEscape(dbUUID)), input, nil); err != nil {
		return fmt.Errorf("bulk updating database env vars %s: %w", dbUUID, err)
	}
	return nil
}

// BulkUpdateServiceEnvVars performs a bulk update of environment variables on a service.
func (c *Client) BulkUpdateServiceEnvVars(ctx context.Context, svcUUID string, input BulkEnvVarInput) error {
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/services/%s/envs/bulk", url.PathEscape(svcUUID)), input, nil); err != nil {
		return fmt.Errorf("bulk updating service env vars %s: %w", svcUUID, err)
	}
	return nil
}

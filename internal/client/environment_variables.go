package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// EnvironmentVariable represents a single environment variable from the Coolify API.
type EnvironmentVariable struct {
	UUID      string `json:"uuid,omitempty"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsPreview bool   `json:"is_preview"`
	IsBuild   bool   `json:"is_buildtime"`
}

// applicationEnvVarInput includes is_buildtime (only valid for applications).
type applicationEnvVarInput struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsPreview bool   `json:"is_preview"`
	IsBuild   *bool  `json:"is_buildtime,omitempty"`
}

// envVarInput is the payload for service/database env var mutations.
type envVarInput struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsPreview bool   `json:"is_preview"`
}

// CreateEnvVarResponse is the response from creating an environment variable.
type CreateEnvVarResponse struct {
	UUID string `json:"uuid"`
}

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

// ---------------------------------------------------------------------------
// Unified environment variable methods (parentType = "applications" | "services" | "databases")
// ---------------------------------------------------------------------------

func envPath(parentType, parentUUID string) string {
	return fmt.Sprintf("/api/v1/%s/%s/envs", parentType, url.PathEscape(parentUUID))
}

// CreateEnvVar creates an environment variable on a parent resource.
// createIsBuild is only sent for applications (pass nil for services/databases).
func (c *Client) CreateEnvVar(ctx context.Context, parentType, parentUUID string, ev EnvironmentVariable, createIsBuild *bool) (*CreateEnvVarResponse, error) {
	if err := validateParentType(parentType); err != nil {
		return nil, err
	}
	var input interface{}
	if parentType == "applications" {
		input = applicationEnvVarInput{Key: ev.Key, Value: ev.Value, IsPreview: ev.IsPreview, IsBuild: createIsBuild}
	} else {
		input = envVarInput{Key: ev.Key, Value: ev.Value, IsPreview: ev.IsPreview}
	}
	var r CreateEnvVarResponse
	path := envPath(parentType, parentUUID)
	if err := c.doWithStatus(ctx, http.MethodPost, path, input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating %s env var %s: %w", parentType, parentUUID, err)
	}
	c.listCache.invalidate(path)
	return &r, nil
}

// ListEnvVars lists all environment variables for a parent resource.
func (c *Client) ListEnvVars(ctx context.Context, parentType, parentUUID string) ([]EnvironmentVariable, error) {
	if err := validateParentType(parentType); err != nil {
		return nil, err
	}
	var v []EnvironmentVariable
	path := envPath(parentType, parentUUID)
	if err := c.doCachedList(ctx, path, &v); err != nil {
		return nil, fmt.Errorf("listing %s env vars %s: %w", parentType, parentUUID, err)
	}
	return v, nil
}

// UpdateEnvVar updates an environment variable on a parent resource.
// For applications, is_buildtime is included in the payload.
func (c *Client) UpdateEnvVar(ctx context.Context, parentType, parentUUID string, ev EnvironmentVariable) error {
	if err := validateParentType(parentType); err != nil {
		return err
	}
	var input interface{}
	if parentType == "applications" {
		input = applicationEnvVarInput{Key: ev.Key, Value: ev.Value, IsPreview: ev.IsPreview, IsBuild: &ev.IsBuild}
	} else {
		input = envVarInput{Key: ev.Key, Value: ev.Value, IsPreview: ev.IsPreview}
	}
	path := envPath(parentType, parentUUID)
	if err := c.do(ctx, http.MethodPatch, path, input, nil); err != nil {
		return fmt.Errorf("updating %s env var %s: %w", parentType, parentUUID, err)
	}
	c.listCache.invalidate(path)
	return nil
}

// DeleteEnvVar deletes a single environment variable from a parent resource.
func (c *Client) DeleteEnvVar(ctx context.Context, parentType, parentUUID, envUUID string) error {
	if err := validateParentType(parentType); err != nil {
		return err
	}
	deletePath := fmt.Sprintf("/api/v1/%s/%s/envs/%s", parentType, url.PathEscape(parentUUID), url.PathEscape(envUUID))
	if err := c.do(ctx, http.MethodDelete, deletePath, nil, nil); err != nil {
		return fmt.Errorf("deleting %s env var %s/%s: %w", parentType, parentUUID, envUUID, err)
	}
	c.listCache.invalidate(envPath(parentType, parentUUID))
	return nil
}

// BulkUpdateEnvVars performs a bulk update of environment variables on a parent resource.
func (c *Client) BulkUpdateEnvVars(ctx context.Context, parentType, parentUUID string, input BulkEnvVarInput) error {
	if err := validateParentType(parentType); err != nil {
		return err
	}
	bulkPath := envPath(parentType, parentUUID) + "/bulk"
	if err := c.do(ctx, http.MethodPatch, bulkPath, input, nil); err != nil {
		return fmt.Errorf("bulk updating %s env vars %s: %w", parentType, parentUUID, err)
	}
	c.listCache.invalidate(envPath(parentType, parentUUID))
	return nil
}

// ---------------------------------------------------------------------------
// Legacy wrappers (delegate to unified methods; kept for backward compatibility
// with existing tests and spectest coverage references)
// ---------------------------------------------------------------------------

func (c *Client) CreateApplicationEnvVar(ctx context.Context, appUUID string, ev EnvironmentVariable, createIsBuild *bool) (*CreateEnvVarResponse, error) {
	return c.CreateEnvVar(ctx, "applications", appUUID, ev, createIsBuild)
}
func (c *Client) ListApplicationEnvVars(ctx context.Context, appUUID string) ([]EnvironmentVariable, error) {
	return c.ListEnvVars(ctx, "applications", appUUID)
}
func (c *Client) UpdateApplicationEnvVar(ctx context.Context, appUUID string, ev EnvironmentVariable) error {
	return c.UpdateEnvVar(ctx, "applications", appUUID, ev)
}
func (c *Client) DeleteApplicationEnvVar(ctx context.Context, appUUID, envUUID string) error {
	return c.DeleteEnvVar(ctx, "applications", appUUID, envUUID)
}
func (c *Client) CreateServiceEnvVar(ctx context.Context, svcUUID string, ev EnvironmentVariable) (*CreateEnvVarResponse, error) {
	return c.CreateEnvVar(ctx, "services", svcUUID, ev, nil)
}
func (c *Client) ListServiceEnvVars(ctx context.Context, svcUUID string) ([]EnvironmentVariable, error) {
	return c.ListEnvVars(ctx, "services", svcUUID)
}
func (c *Client) UpdateServiceEnvVar(ctx context.Context, svcUUID string, ev EnvironmentVariable) error {
	return c.UpdateEnvVar(ctx, "services", svcUUID, ev)
}
func (c *Client) DeleteServiceEnvVar(ctx context.Context, svcUUID, envUUID string) error {
	return c.DeleteEnvVar(ctx, "services", svcUUID, envUUID)
}
func (c *Client) CreateDatabaseEnvVar(ctx context.Context, dbUUID string, ev EnvironmentVariable) (*CreateEnvVarResponse, error) {
	return c.CreateEnvVar(ctx, "databases", dbUUID, ev, nil)
}
func (c *Client) ListDatabaseEnvVars(ctx context.Context, dbUUID string) ([]EnvironmentVariable, error) {
	return c.ListEnvVars(ctx, "databases", dbUUID)
}
func (c *Client) UpdateDatabaseEnvVar(ctx context.Context, dbUUID string, ev EnvironmentVariable) error {
	return c.UpdateEnvVar(ctx, "databases", dbUUID, ev)
}
func (c *Client) DeleteDatabaseEnvVar(ctx context.Context, dbUUID, envUUID string) error {
	return c.DeleteEnvVar(ctx, "databases", dbUUID, envUUID)
}
func (c *Client) BulkUpdateAppEnvVars(ctx context.Context, appUUID string, input BulkEnvVarInput) error {
	return c.BulkUpdateEnvVars(ctx, "applications", appUUID, input)
}
func (c *Client) BulkUpdateDatabaseEnvVars(ctx context.Context, dbUUID string, input BulkEnvVarInput) error {
	return c.BulkUpdateEnvVars(ctx, "databases", dbUUID, input)
}
func (c *Client) BulkUpdateServiceEnvVars(ctx context.Context, svcUUID string, input BulkEnvVarInput) error {
	return c.BulkUpdateEnvVars(ctx, "services", svcUUID, input)
}

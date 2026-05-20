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

// PreferNonPreviewEnvVarsByKey collapses duplicate preview and non-preview rows
// by key, preferring the non-preview value when both exist.
func PreferNonPreviewEnvVarsByKey(envs []EnvironmentVariable) map[string]EnvironmentVariable {
	vars := make(map[string]EnvironmentVariable, len(envs))
	for _, ev := range envs {
		current, ok := vars[ev.Key]
		if ok && !current.IsPreview && ev.IsPreview {
			continue
		}
		vars[ev.Key] = ev
	}
	return vars
}

// PreserveEnvVarValue keeps the previous Terraform value when the API hides a
// sensitive value by returning an empty string.
func PreserveEnvVarValue(current, prior string) string {
	if current != "" || prior == "" {
		return current
	}
	return prior
}

// FindEnvVarByUUID returns the matching env var from a list along with whether
// it was found.
func FindEnvVarByUUID(envs []EnvironmentVariable, uuid string) (EnvironmentVariable, bool) {
	for _, ev := range envs {
		if ev.UUID == uuid {
			return ev, true
		}
	}
	return EnvironmentVariable{}, false
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

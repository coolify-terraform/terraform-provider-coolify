package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// EnvironmentVariable represents a single environment variable from the Coolify API.
type EnvironmentVariable struct {
	UUID        string `json:"uuid,omitempty"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	IsPreview   bool   `json:"is_preview"`
	IsBuild     bool   `json:"is_buildtime"`
	IsRuntime   bool   `json:"is_runtime"`
	IsLiteral   bool   `json:"is_literal"`
	IsMultiline bool   `json:"is_multiline"`
	Comment     string `json:"comment,omitempty"`
}

// EnvVarWriteOpts holds optional fields for create/update payloads.
// A nil pointer omits the JSON key so Coolify can apply its default.
// Application-only: IsBuild, IsRuntime. All parents: IsLiteral, IsMultiline, Comment.
type EnvVarWriteOpts struct {
	IsBuild     *bool
	IsRuntime   *bool
	IsLiteral   *bool
	IsMultiline *bool
	Comment     *string
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

// applicationEnvVarInput is the write payload for application envs.
// Coolify accepts is_runtime and is_buildtime only on applications.
type applicationEnvVarInput struct {
	Key         string  `json:"key"`
	Value       string  `json:"value"`
	IsPreview   bool    `json:"is_preview"`
	IsBuild     *bool   `json:"is_buildtime,omitempty"`
	IsRuntime   *bool   `json:"is_runtime,omitempty"`
	IsLiteral   *bool   `json:"is_literal,omitempty"`
	IsMultiline *bool   `json:"is_multiline,omitempty"`
	Comment     *string `json:"comment,omitempty"`
}

// envVarInput is the write payload for service/database envs.
// Coolify does not accept is_buildtime / is_runtime on these parents.
type envVarInput struct {
	Key         string  `json:"key"`
	Value       string  `json:"value"`
	IsPreview   bool    `json:"is_preview"`
	IsLiteral   *bool   `json:"is_literal,omitempty"`
	IsMultiline *bool   `json:"is_multiline,omitempty"`
	Comment     *string `json:"comment,omitempty"`
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

func buildEnvWriteInput(parentType string, key, value string, isPreview bool, opts *EnvVarWriteOpts) interface{} {
	if opts == nil {
		opts = &EnvVarWriteOpts{}
	}
	if parentType == "applications" {
		return applicationEnvVarInput{
			Key:         key,
			Value:       value,
			IsPreview:   isPreview,
			IsBuild:     opts.IsBuild,
			IsRuntime:   opts.IsRuntime,
			IsLiteral:   opts.IsLiteral,
			IsMultiline: opts.IsMultiline,
			Comment:     opts.Comment,
		}
	}
	return envVarInput{
		Key:         key,
		Value:       value,
		IsPreview:   isPreview,
		IsLiteral:   opts.IsLiteral,
		IsMultiline: opts.IsMultiline,
		Comment:     opts.Comment,
	}
}

// CreateEnvVar creates an environment variable on a parent resource.
// Pass opts for optional flags; nil pointers are omitted from the JSON body.
func (c *Client) CreateEnvVar(ctx context.Context, parentType, parentUUID string, ev EnvironmentVariable, opts *EnvVarWriteOpts) (*CreateEnvVarResponse, error) {
	if err := validateParentType(parentType); err != nil {
		return nil, err
	}
	input := buildEnvWriteInput(parentType, ev.Key, ev.Value, ev.IsPreview, opts)
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
// opts should carry the flags to send; application paths may include IsBuild/IsRuntime.
func (c *Client) UpdateEnvVar(ctx context.Context, parentType, parentUUID string, ev EnvironmentVariable, opts *EnvVarWriteOpts) error {
	if err := validateParentType(parentType); err != nil {
		return err
	}
	if opts == nil {
		// Back-compat: populate write opts from the full entity so application
		// omit-as-false fields (is_literal) are not silently cleared.
		b, r, l, m := ev.IsBuild, ev.IsRuntime, ev.IsLiteral, ev.IsMultiline
		c := ev.Comment
		opts = &EnvVarWriteOpts{
			IsBuild:     &b,
			IsRuntime:   &r,
			IsLiteral:   &l,
			IsMultiline: &m,
			Comment:     &c,
		}
	}
	input := buildEnvWriteInput(parentType, ev.Key, ev.Value, ev.IsPreview, opts)
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

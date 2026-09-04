package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// SharedEnvironmentVariable is a Coolify shared env (team/project/environment/server).
type SharedEnvironmentVariable struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid,omitempty"`
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"`
	IsLiteral   bool   `json:"is_literal"`
	IsMultiline bool   `json:"is_multiline"`
	IsShownOnce bool   `json:"is_shown_once"`
	Comment     string `json:"comment,omitempty"`
}

type SharedEnvInput struct {
	Key         string `json:"key,omitempty"`
	Value       string `json:"value,omitempty"`
	IsLiteral   *bool  `json:"is_literal,omitempty"`
	IsMultiline *bool  `json:"is_multiline,omitempty"`
	IsShownOnce *bool  `json:"is_shown_once,omitempty"`
	Comment     string `json:"comment,omitempty"`
}

func sharedEnvBase(scope, projectUUID, environment, serverUUID string) (string, error) {
	switch scope {
	case "team":
		return "/api/v1/team/envs", nil
	case "project":
		if projectUUID == "" {
			return "", fmt.Errorf("project_uuid is required for project scope")
		}
		return fmt.Sprintf("/api/v1/projects/%s/envs", url.PathEscape(projectUUID)), nil
	case "environment":
		if projectUUID == "" || environment == "" {
			return "", fmt.Errorf("project_uuid and environment are required for environment scope")
		}
		return fmt.Sprintf("/api/v1/projects/%s/environments/%s/envs", url.PathEscape(projectUUID), url.PathEscape(environment)), nil
	case "server":
		if serverUUID == "" {
			return "", fmt.Errorf("server_uuid is required for server scope")
		}
		return fmt.Sprintf("/api/v1/servers/%s/envs", url.PathEscape(serverUUID)), nil
	default:
		return "", fmt.Errorf("unsupported shared env scope %q", scope)
	}
}

func (c *Client) ListSharedEnvs(ctx context.Context, scope, projectUUID, environment, serverUUID string) ([]SharedEnvironmentVariable, error) {
	base, err := sharedEnvBase(scope, projectUUID, environment, serverUUID)
	if err != nil {
		return nil, err
	}
	var r []SharedEnvironmentVariable
	if err := c.doCachedList(ctx, base, &r); err != nil {
		return nil, fmt.Errorf("listing shared envs (%s): %w", scope, err)
	}
	return r, nil
}

func (c *Client) CreateSharedEnv(ctx context.Context, scope, projectUUID, environment, serverUUID string, input SharedEnvInput) (*SharedEnvironmentVariable, error) {
	base, err := sharedEnvBase(scope, projectUUID, environment, serverUUID)
	if err != nil {
		return nil, err
	}
	var r SharedEnvironmentVariable
	if err := c.doWithStatus(ctx, http.MethodPost, base, input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating shared env %s: %w", input.Key, err)
	}
	c.listCache.invalidate(base)
	return &r, nil
}

func (c *Client) UpdateSharedEnv(ctx context.Context, scope, projectUUID, environment, serverUUID, envID string, input SharedEnvInput) (*SharedEnvironmentVariable, error) {
	base, err := sharedEnvBase(scope, projectUUID, environment, serverUUID)
	if err != nil {
		return nil, err
	}
	var r SharedEnvironmentVariable
	if err := c.do(ctx, http.MethodPatch, base+"/"+url.PathEscape(envID), input, &r); err != nil {
		return nil, fmt.Errorf("updating shared env %s: %w", envID, err)
	}
	c.listCache.invalidate(base)
	return &r, nil
}

func (c *Client) DeleteSharedEnv(ctx context.Context, scope, projectUUID, environment, serverUUID, envID string) error {
	base, err := sharedEnvBase(scope, projectUUID, environment, serverUUID)
	if err != nil {
		return err
	}
	if err := c.do(ctx, http.MethodDelete, base+"/"+url.PathEscape(envID), nil, nil); err != nil {
		return fmt.Errorf("deleting shared env %s: %w", envID, err)
	}
	c.listCache.invalidate(base)
	return nil
}

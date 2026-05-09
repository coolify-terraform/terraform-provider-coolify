package client

import (
	"context"
	"fmt"
	"net/http"
)

type EnvironmentVariable struct {
	UUID      string `json:"uuid"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsPreview bool   `json:"is_preview"`
	IsBuild   bool   `json:"is_build_time"`
}
type CreateEnvVarResponse struct {
	UUID string `json:"uuid"`
}

func (c *Client) CreateApplicationEnvVar(ctx context.Context, appUUID string, ev EnvironmentVariable) (*CreateEnvVarResponse, error) {
	var r CreateEnvVarResponse
	if err := c.doWithStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/applications/%s/envs", appUUID), ev, &r, http.StatusCreated); err != nil {
		return nil, err
	}
	return &r, nil
}
func (c *Client) ListApplicationEnvVars(ctx context.Context, appUUID string) ([]EnvironmentVariable, error) {
	var v []EnvironmentVariable
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/applications/%s/envs", appUUID), nil, &v); err != nil {
		return nil, err
	}
	return v, nil
}
func (c *Client) UpdateApplicationEnvVar(ctx context.Context, appUUID string, ev EnvironmentVariable) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/applications/%s/envs", appUUID), ev, nil)
}
func (c *Client) DeleteApplicationEnvVar(ctx context.Context, appUUID string, envUUID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/applications/%s/envs/%s", appUUID, envUUID), nil, nil)
}
func (c *Client) CreateServiceEnvVar(ctx context.Context, svcUUID string, ev EnvironmentVariable) (*CreateEnvVarResponse, error) {
	var r CreateEnvVarResponse
	if err := c.doWithStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/services/%s/envs", svcUUID), ev, &r, http.StatusCreated); err != nil {
		return nil, err
	}
	return &r, nil
}
func (c *Client) ListServiceEnvVars(ctx context.Context, svcUUID string) ([]EnvironmentVariable, error) {
	var v []EnvironmentVariable
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/services/%s/envs", svcUUID), nil, &v); err != nil {
		return nil, err
	}
	return v, nil
}
func (c *Client) UpdateServiceEnvVar(ctx context.Context, svcUUID string, ev EnvironmentVariable) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/services/%s/envs", svcUUID), ev, nil)
}
func (c *Client) DeleteServiceEnvVar(ctx context.Context, svcUUID string, envUUID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/services/%s/envs/%s", svcUUID, envUUID), nil, nil)
}

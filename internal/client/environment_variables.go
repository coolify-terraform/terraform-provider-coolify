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

func (c *Client) CreateApplicationEnvVar(appUUID string, ev EnvironmentVariable) (*CreateEnvVarResponse, error) {
	var r CreateEnvVarResponse
	if err := c.doWithStatus(context.Background(), http.MethodPost, fmt.Sprintf("/api/v1/applications/%s/envs", appUUID), ev, &r, http.StatusCreated); err != nil {
		return nil, err
	}
	return &r, nil
}
func (c *Client) ListApplicationEnvVars(appUUID string) ([]EnvironmentVariable, error) {
	var v []EnvironmentVariable
	if err := c.do(context.Background(), http.MethodGet, fmt.Sprintf("/api/v1/applications/%s/envs", appUUID), nil, &v); err != nil {
		return nil, err
	}
	return v, nil
}
func (c *Client) UpdateApplicationEnvVar(appUUID string, ev EnvironmentVariable) error {
	return c.do(context.Background(), http.MethodPatch, fmt.Sprintf("/api/v1/applications/%s/envs", appUUID), ev, nil)
}
func (c *Client) DeleteApplicationEnvVar(appUUID string, envUUID string) error {
	return c.do(context.Background(), http.MethodDelete, fmt.Sprintf("/api/v1/applications/%s/envs/%s", appUUID, envUUID), nil, nil)
}
func (c *Client) CreateServiceEnvVar(svcUUID string, ev EnvironmentVariable) (*CreateEnvVarResponse, error) {
	var r CreateEnvVarResponse
	if err := c.doWithStatus(context.Background(), http.MethodPost, fmt.Sprintf("/api/v1/services/%s/envs", svcUUID), ev, &r, http.StatusCreated); err != nil {
		return nil, err
	}
	return &r, nil
}
func (c *Client) ListServiceEnvVars(svcUUID string) ([]EnvironmentVariable, error) {
	var v []EnvironmentVariable
	if err := c.do(context.Background(), http.MethodGet, fmt.Sprintf("/api/v1/services/%s/envs", svcUUID), nil, &v); err != nil {
		return nil, err
	}
	return v, nil
}
func (c *Client) UpdateServiceEnvVar(svcUUID string, ev EnvironmentVariable) error {
	return c.do(context.Background(), http.MethodPatch, fmt.Sprintf("/api/v1/services/%s/envs", svcUUID), ev, nil)
}
func (c *Client) DeleteServiceEnvVar(svcUUID string, envUUID string) error {
	return c.do(context.Background(), http.MethodDelete, fmt.Sprintf("/api/v1/services/%s/envs/%s", svcUUID, envUUID), nil, nil)
}

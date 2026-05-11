package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type CloudToken struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Token    string `json:"token,omitempty"`
}

type CreateCloudTokenInput struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Token    string `json:"token"`
}

type UpdateCloudTokenInput struct {
	Name  *string `json:"name,omitempty"`
	Token *string `json:"token,omitempty"`
}

func (c *Client) ListCloudTokens(ctx context.Context) ([]CloudToken, error) {
	var r []CloudToken
	if err := c.do(ctx, http.MethodGet, "/api/v1/cloud-tokens", nil, &r); err != nil {
		return nil, fmt.Errorf("listing cloud tokens: %w", err)
	}
	return r, nil
}

func (c *Client) GetCloudToken(ctx context.Context, uuid string) (*CloudToken, error) {
	var r CloudToken
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/cloud-tokens/%s", url.PathEscape(uuid)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting cloud token %s: %w", uuid, err)
	}
	return &r, nil
}

func (c *Client) CreateCloudToken(ctx context.Context, input CreateCloudTokenInput) (*CloudToken, error) {
	var r CloudToken
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/cloud-tokens", input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating cloud token: %w", err)
	}
	return &r, nil
}

func (c *Client) UpdateCloudToken(ctx context.Context, uuid string, input UpdateCloudTokenInput) (*CloudToken, error) {
	var r CloudToken
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/cloud-tokens/%s", url.PathEscape(uuid)), input, &r); err != nil {
		return nil, fmt.Errorf("updating cloud token %s: %w", uuid, err)
	}
	return &r, nil
}

func (c *Client) DeleteCloudToken(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/cloud-tokens/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting cloud token %s: %w", uuid, err)
	}
	return nil
}

func (c *Client) ValidateCloudToken(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/cloud-tokens/%s/validate", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("validating cloud token %s: %w", uuid, err)
	}
	return nil
}

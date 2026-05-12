package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type PrivateKey struct {
	UUID         string `json:"uuid,omitempty"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	PrivateKey   string `json:"private_key"`
	PublicKey    string `json:"public_key,omitempty"`
	Fingerprint  string `json:"fingerprint,omitempty"`
	IsGitRelated bool   `json:"is_git_related"`
}
type CreatePrivateKeyInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	PrivateKey  string `json:"private_key"`
}
type UpdatePrivateKeyInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	PrivateKey  *string `json:"private_key,omitempty"`
}

func (c *Client) ListPrivateKeys(ctx context.Context) ([]PrivateKey, error) {
	var k []PrivateKey
	if err := c.do(ctx, http.MethodGet, "/api/v1/security/keys", nil, &k); err != nil {
		return nil, fmt.Errorf("listing private keys: %w", err)
	}
	return k, nil
}
func (c *Client) GetPrivateKey(ctx context.Context, uuid string) (*PrivateKey, error) {
	var k PrivateKey
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/security/keys/%s", url.PathEscape(uuid)), nil, &k); err != nil {
		return nil, fmt.Errorf("getting private key %s: %w", uuid, err)
	}
	return &k, nil
}
func (c *Client) CreatePrivateKey(ctx context.Context, input CreatePrivateKeyInput) (*PrivateKey, error) {
	var k PrivateKey
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/security/keys", input, &k, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating private key: %w", err)
	}
	return &k, nil
}
func (c *Client) UpdatePrivateKey(ctx context.Context, uuid string, input UpdatePrivateKeyInput) (*PrivateKey, error) {
	var k PrivateKey
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/security/keys/%s", url.PathEscape(uuid)), input, &k); err != nil {
		return nil, fmt.Errorf("updating private key %s: %w", uuid, err)
	}
	return &k, nil
}
func (c *Client) DeletePrivateKey(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/security/keys/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting private key %s: %w", uuid, err)
	}
	return nil
}

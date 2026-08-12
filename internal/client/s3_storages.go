package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// S3Storage is an S3-compatible storage configuration in Coolify.
// Key and Secret are sensitive; Coolify omits them unless the API token
// has can_read_sensitive permission.
type S3Storage struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Endpoint    string `json:"endpoint"`
	Bucket      string `json:"bucket"`
	Region      string `json:"region"`
	Key         string `json:"key,omitempty"`
	Secret      string `json:"secret,omitempty"`
	IsUsable    *bool  `json:"is_usable,omitempty"`
}

// CreateS3StorageInput is the payload for creating S3 storage.
// Required: name, endpoint, bucket, region, key, secret.
type CreateS3StorageInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Endpoint    string `json:"endpoint"`
	Bucket      string `json:"bucket"`
	Region      string `json:"region"`
	Key         string `json:"key"`
	Secret      string `json:"secret"`
	IsUsable    *bool  `json:"is_usable,omitempty"`
}

// UpdateS3StorageInput is the payload for updating S3 storage.
// All fields are optional; only non-nil fields are sent.
type UpdateS3StorageInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Endpoint    *string `json:"endpoint,omitempty"`
	Bucket      *string `json:"bucket,omitempty"`
	Region      *string `json:"region,omitempty"`
	Key         *string `json:"key,omitempty"`
	Secret      *string `json:"secret,omitempty"`
	IsUsable    *bool   `json:"is_usable,omitempty"`
}

// ListS3Storages returns all S3 storages for the authenticated team.
// Requires Coolify >= v4.3.0.
func (c *Client) ListS3Storages(ctx context.Context) ([]S3Storage, error) {
	var r []S3Storage
	if err := c.do(ctx, http.MethodGet, "/api/v1/s3-storages", nil, &r); err != nil {
		return nil, fmt.Errorf("listing s3 storages: %w", err)
	}
	return r, nil
}

// GetS3Storage returns a single S3 storage by UUID.
// Requires Coolify >= v4.3.0.
func (c *Client) GetS3Storage(ctx context.Context, uuid string) (*S3Storage, error) {
	var r S3Storage
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/s3-storages/%s", url.PathEscape(uuid)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting s3 storage %s: %w", uuid, err)
	}
	return &r, nil
}

// CreateS3Storage creates a new S3 storage configuration.
// Coolify returns 201 with {uuid: "..."}. Requires Coolify >= v4.3.0.
func (c *Client) CreateS3Storage(ctx context.Context, input CreateS3StorageInput) (*S3Storage, error) {
	var r S3Storage
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/s3-storages", input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating s3 storage: %w", err)
	}
	return &r, nil
}

// UpdateS3Storage updates an existing S3 storage by UUID.
// Requires Coolify >= v4.3.0.
func (c *Client) UpdateS3Storage(ctx context.Context, uuid string, input UpdateS3StorageInput) (*S3Storage, error) {
	var r S3Storage
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/s3-storages/%s", url.PathEscape(uuid)), input, &r); err != nil {
		return nil, fmt.Errorf("updating s3 storage %s: %w", uuid, err)
	}
	return &r, nil
}

// DeleteS3Storage deletes an S3 storage by UUID.
// Requires Coolify >= v4.3.0.
func (c *Client) DeleteS3Storage(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/s3-storages/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting s3 storage %s: %w", uuid, err)
	}
	return nil
}

// ValidateS3Storage tests the S3 connection and updates is_usable on the server.
// Requires Coolify >= v4.3.0.
func (c *Client) ValidateS3Storage(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/s3-storages/%s/validate", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("validating s3 storage %s: %w", uuid, err)
	}
	return nil
}

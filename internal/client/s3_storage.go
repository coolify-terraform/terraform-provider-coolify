package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type S3Storage struct {
	ID          int    `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Endpoint    string `json:"endpoint"`
	Bucket      string `json:"bucket"`
	Region      string `json:"region"`
	AccessKey   string `json:"access_key"`
	SecretKey   string `json:"secret_key"`
}

type CreateS3StorageInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Endpoint    string `json:"endpoint"`
	Bucket      string `json:"bucket"`
	Region      string `json:"region"`
	AccessKey   string `json:"access_key"`
	SecretKey   string `json:"secret_key"`
}

type UpdateS3StorageInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Endpoint    *string `json:"endpoint,omitempty"`
	Bucket      *string `json:"bucket,omitempty"`
	Region      *string `json:"region,omitempty"`
	AccessKey   *string `json:"access_key,omitempty"`
	SecretKey   *string `json:"secret_key,omitempty"`
}

func (c *Client) ListS3Storages(ctx context.Context) ([]S3Storage, error) {
	var s []S3Storage
	if err := c.do(ctx, http.MethodGet, "/api/v1/storages", nil, &s); err != nil {
		return nil, fmt.Errorf("listing s3 storages: %w", err)
	}
	return s, nil
}

func (c *Client) GetS3Storage(ctx context.Context, uuid string) (*S3Storage, error) {
	var s S3Storage
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/storages/%s", url.PathEscape(uuid)), nil, &s); err != nil {
		return nil, fmt.Errorf("getting s3 storage %s: %w", uuid, err)
	}
	return &s, nil
}

func (c *Client) CreateS3Storage(ctx context.Context, input CreateS3StorageInput) (*S3Storage, error) {
	var s S3Storage
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/storages", input, &s, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating s3 storage: %w", err)
	}
	return &s, nil
}

func (c *Client) UpdateS3Storage(ctx context.Context, uuid string, input UpdateS3StorageInput) (*S3Storage, error) {
	var s S3Storage
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/storages/%s", url.PathEscape(uuid)), input, &s); err != nil {
		return nil, fmt.Errorf("updating s3 storage %s: %w", uuid, err)
	}
	return &s, nil
}

func (c *Client) DeleteS3Storage(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/storages/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting s3 storage %s: %w", uuid, err)
	}
	return nil
}

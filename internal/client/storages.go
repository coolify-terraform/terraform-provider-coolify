package client

import (
	"context"
	"fmt"
	"net/http"
)

// Storage represents a persistent storage volume in Coolify.
type Storage struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	HostPath  string `json:"host_path,omitempty"`
}

// CreateStorageInput is the payload for creating a new persistent storage.
type CreateStorageInput struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	HostPath  string `json:"host_path,omitempty"`
}

// CreateStorageResponse is the response from creating a persistent storage.
type CreateStorageResponse struct {
	UUID string `json:"uuid"`
}

// UpdateStorageInput is the payload for updating a persistent storage.
// All fields are optional; only non-nil fields are sent.
type UpdateStorageInput struct {
	UUID      *string `json:"uuid,omitempty"`
	Name      *string `json:"name,omitempty"`
	MountPath *string `json:"mount_path,omitempty"`
	HostPath  *string `json:"host_path,omitempty"`
}

// ListStorages lists all persistent storages for a parent resource.
// parentType must be "applications", "databases", or "services".
func (c *Client) ListStorages(ctx context.Context, parentType, parentUUID string) ([]Storage, error) {
	var v []Storage
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/%s/%s/storages", parentType, parentUUID), nil, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// CreateStorage creates a new persistent storage on a parent resource.
// The API returns 201 on success.
func (c *Client) CreateStorage(ctx context.Context, parentType, parentUUID string, input CreateStorageInput) (*CreateStorageResponse, error) {
	var r CreateStorageResponse
	if err := c.doWithStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/%s/%s/storages", parentType, parentUUID), input, &r, http.StatusCreated); err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdateStorage updates a persistent storage on a parent resource.
// The API uses PATCH to the parent storages path (not per-storage).
func (c *Client) UpdateStorage(ctx context.Context, parentType, parentUUID string, input UpdateStorageInput) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/%s/%s/storages", parentType, parentUUID), input, nil)
}

// DeleteStorage deletes a persistent storage from a parent resource.
func (c *Client) DeleteStorage(ctx context.Context, parentType, parentUUID, storageUUID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/%s/%s/storages/%s", parentType, parentUUID, storageUUID), nil, nil)
}

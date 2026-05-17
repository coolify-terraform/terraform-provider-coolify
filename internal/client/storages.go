package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Storage represents a persistent storage volume in Coolify.
type Storage struct {
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	MountPath    string `json:"mount_path"`
	HostPath     string `json:"host_path,omitempty"`
	ResourceUUID string `json:"resource_uuid,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
}

// storageListResponse wraps the API response which nests storages by type.
type storageListResponse struct {
	PersistentStorages []Storage `json:"persistent_storages"`
	FileStorages       []Storage `json:"file_storages"`
}

// CreateStorageInput is the payload for creating a new persistent storage.
type CreateStorageInput struct {
	Type         string `json:"type"`
	ResourceUUID string `json:"resource_uuid,omitempty"`
	Name         string `json:"name"`
	MountPath    string `json:"mount_path"`
	HostPath     string `json:"host_path,omitempty"`
}

// CreateStorageResponse is the response from creating a persistent storage.
type CreateStorageResponse struct {
	UUID string `json:"uuid"`
}

// UpdateStorageInput is the payload for updating a persistent storage.
// All fields are optional; only non-nil fields are sent.
type UpdateStorageInput struct {
	UUID      *string `json:"uuid,omitempty"`
	Type      string  `json:"type"`
	Name      *string `json:"name,omitempty"`
	MountPath *string `json:"mount_path,omitempty"`
	HostPath  *string `json:"host_path,omitempty"`
}

// ListStorages lists all persistent storages for a parent resource.
// parentType must be "applications", "databases", or "services".
func (c *Client) ListStorages(ctx context.Context, parentType, parentUUID string) ([]Storage, error) {
	if err := validateParentType(parentType); err != nil {
		return nil, err
	}
	var v storageListResponse
	path := fmt.Sprintf("/api/v1/%s/%s/storages", parentType, url.PathEscape(parentUUID))
	if err := c.doCachedList(ctx, path, &v); err != nil {
		return nil, fmt.Errorf("listing storages for %s %s: %w", parentType, parentUUID, err)
	}
	return append(v.PersistentStorages, v.FileStorages...), nil
}

// CreateStorage creates a new persistent storage on a parent resource.
// The API returns 201 on success.
func (c *Client) CreateStorage(ctx context.Context, parentType, parentUUID string, input CreateStorageInput) (*CreateStorageResponse, error) {
	if err := validateParentType(parentType); err != nil {
		return nil, err
	}
	var r CreateStorageResponse
	listPath := fmt.Sprintf("/api/v1/%s/%s/storages", parentType, url.PathEscape(parentUUID))
	if err := c.doWithStatus(ctx, http.MethodPost, listPath, input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating storage for %s %s: %w", parentType, parentUUID, err)
	}
	c.listCache.invalidate(listPath)
	return &r, nil
}

// UpdateStorage updates a persistent storage on a parent resource.
// The API uses PATCH to the parent storages path (not per-storage).
func (c *Client) UpdateStorage(ctx context.Context, parentType, parentUUID string, input UpdateStorageInput) error {
	if err := validateParentType(parentType); err != nil {
		return err
	}
	listPath := fmt.Sprintf("/api/v1/%s/%s/storages", parentType, url.PathEscape(parentUUID))
	if err := c.do(ctx, http.MethodPatch, listPath, input, nil); err != nil {
		return fmt.Errorf("updating storage for %s %s: %w", parentType, parentUUID, err)
	}
	c.listCache.invalidate(listPath)
	return nil
}

// DeleteStorage deletes a persistent storage from a parent resource.
func (c *Client) DeleteStorage(ctx context.Context, parentType, parentUUID, storageUUID string) error {
	if err := validateParentType(parentType); err != nil {
		return err
	}
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/%s/%s/storages/%s", parentType, url.PathEscape(parentUUID), url.PathEscape(storageUUID)), nil, nil); err != nil {
		return fmt.Errorf("deleting storage %s for %s %s: %w", storageUUID, parentType, parentUUID, err)
	}
	c.listCache.invalidate(fmt.Sprintf("/api/v1/%s/%s/storages", parentType, url.PathEscape(parentUUID)))
	return nil
}

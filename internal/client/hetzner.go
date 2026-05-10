package client

import (
	"context"
	"fmt"
	"net/http"
)

// HetznerImage represents a Hetzner cloud image.
type HetznerImage struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// HetznerLocation represents a Hetzner datacenter location.
type HetznerLocation struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	City        string `json:"city"`
	Country     string `json:"country"`
}

// HetznerServerType represents a Hetzner server type.
type HetznerServerType struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Cores       int64  `json:"cores"`
	Memory      int64  `json:"memory"`
	Disk        int64  `json:"disk"`
}

// HetznerSSHKey represents a Hetzner SSH key.
type HetznerSSHKey struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
}

// ListHetznerImages returns all Hetzner cloud images.
func (c *Client) ListHetznerImages(ctx context.Context) ([]HetznerImage, error) {
	var images []HetznerImage
	if err := c.do(ctx, http.MethodGet, "/api/v1/hetzner/images", nil, &images); err != nil {
		return nil, fmt.Errorf("listing hetzner images: %w", err)
	}
	return images, nil
}

// ListHetznerLocations returns all Hetzner datacenter locations.
func (c *Client) ListHetznerLocations(ctx context.Context) ([]HetznerLocation, error) {
	var locations []HetznerLocation
	if err := c.do(ctx, http.MethodGet, "/api/v1/hetzner/locations", nil, &locations); err != nil {
		return nil, fmt.Errorf("listing hetzner locations: %w", err)
	}
	return locations, nil
}

// ListHetznerServerTypes returns all Hetzner server types.
func (c *Client) ListHetznerServerTypes(ctx context.Context) ([]HetznerServerType, error) {
	var types []HetznerServerType
	if err := c.do(ctx, http.MethodGet, "/api/v1/hetzner/server-types", nil, &types); err != nil {
		return nil, fmt.Errorf("listing hetzner server types: %w", err)
	}
	return types, nil
}

// ListHetznerSSHKeys returns all Hetzner SSH keys.
func (c *Client) ListHetznerSSHKeys(ctx context.Context) ([]HetznerSSHKey, error) {
	var keys []HetznerSSHKey
	if err := c.do(ctx, http.MethodGet, "/api/v1/hetzner/ssh-keys", nil, &keys); err != nil {
		return nil, fmt.Errorf("listing hetzner ssh keys: %w", err)
	}
	return keys, nil
}

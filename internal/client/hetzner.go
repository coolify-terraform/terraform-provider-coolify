package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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

func hetznerPath(base, cloudProviderTokenUUID string) string {
	q := url.Values{}
	q.Set("cloud_provider_token_uuid", cloudProviderTokenUUID)
	return base + "?" + q.Encode()
}

// ListHetznerImages returns all Hetzner cloud images for the given cloud provider token.
func (c *Client) ListHetznerImages(ctx context.Context, cloudProviderTokenUUID string) ([]HetznerImage, error) {
	var images []HetznerImage
	if err := c.do(ctx, http.MethodGet, hetznerPath("/api/v1/hetzner/images", cloudProviderTokenUUID), nil, &images); err != nil {
		return nil, fmt.Errorf("listing hetzner images: %w", err)
	}
	return images, nil
}

// ListHetznerLocations returns all Hetzner datacenter locations for the given cloud provider token.
func (c *Client) ListHetznerLocations(ctx context.Context, cloudProviderTokenUUID string) ([]HetznerLocation, error) {
	var locations []HetznerLocation
	if err := c.do(ctx, http.MethodGet, hetznerPath("/api/v1/hetzner/locations", cloudProviderTokenUUID), nil, &locations); err != nil {
		return nil, fmt.Errorf("listing hetzner locations: %w", err)
	}
	return locations, nil
}

// ListHetznerServerTypes returns all Hetzner server types for the given cloud provider token.
func (c *Client) ListHetznerServerTypes(ctx context.Context, cloudProviderTokenUUID string) ([]HetznerServerType, error) {
	var types []HetznerServerType
	if err := c.do(ctx, http.MethodGet, hetznerPath("/api/v1/hetzner/server-types", cloudProviderTokenUUID), nil, &types); err != nil {
		return nil, fmt.Errorf("listing hetzner server types: %w", err)
	}
	return types, nil
}

// ListHetznerSSHKeys returns all Hetzner SSH keys for the given cloud provider token.
func (c *Client) ListHetznerSSHKeys(ctx context.Context, cloudProviderTokenUUID string) ([]HetznerSSHKey, error) {
	var keys []HetznerSSHKey
	if err := c.do(ctx, http.MethodGet, hetznerPath("/api/v1/hetzner/ssh-keys", cloudProviderTokenUUID), nil, &keys); err != nil {
		return nil, fmt.Errorf("listing hetzner ssh keys: %w", err)
	}
	return keys, nil
}

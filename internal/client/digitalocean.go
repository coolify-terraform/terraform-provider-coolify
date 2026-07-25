package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DigitalOceanRegion is a DigitalOcean region.
type DigitalOceanRegion struct {
	Slug      string   `json:"slug"`
	Name      string   `json:"name"`
	Available bool     `json:"available"`
	Features  []string `json:"features,omitempty"`
}

// DigitalOceanSize is a DigitalOcean droplet size.
type DigitalOceanSize struct {
	Slug         string  `json:"slug"`
	Memory       int64   `json:"memory"`
	VCPUs        int64   `json:"vcpus"`
	Disk         int64   `json:"disk"`
	Transfer     float64 `json:"transfer"`
	PriceMonthly float64 `json:"price_monthly"`
	PriceHourly  float64 `json:"price_hourly"`
	Available    bool    `json:"available"`
}

// DigitalOceanImage is a DigitalOcean image.
type DigitalOceanImage struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Distribution string `json:"distribution"`
	Slug         string `json:"slug"`
	Public       bool   `json:"public"`
	Type         string `json:"type"`
}

// DigitalOceanSSHKey is a DigitalOcean SSH key.
type DigitalOceanSSHKey struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
}

func digitalOceanPath(base, cloudProviderTokenUUID string) string {
	q := url.Values{}
	q.Set("cloud_provider_token_uuid", cloudProviderTokenUUID)
	return base + "?" + q.Encode()
}

// ListDigitalOceanRegions returns regions for a DigitalOcean cloud token (Coolify >= v4.2.0).
func (c *Client) ListDigitalOceanRegions(ctx context.Context, cloudProviderTokenUUID string) ([]DigitalOceanRegion, error) {
	var regions []DigitalOceanRegion
	if err := c.do(ctx, http.MethodGet, digitalOceanPath("/api/v1/digitalocean/regions", cloudProviderTokenUUID), nil, &regions); err != nil {
		return nil, fmt.Errorf("listing digitalocean regions: %w", err)
	}
	return regions, nil
}

// ListDigitalOceanSizes returns droplet sizes for a DigitalOcean cloud token (Coolify >= v4.2.0).
func (c *Client) ListDigitalOceanSizes(ctx context.Context, cloudProviderTokenUUID string) ([]DigitalOceanSize, error) {
	var sizes []DigitalOceanSize
	if err := c.do(ctx, http.MethodGet, digitalOceanPath("/api/v1/digitalocean/sizes", cloudProviderTokenUUID), nil, &sizes); err != nil {
		return nil, fmt.Errorf("listing digitalocean sizes: %w", err)
	}
	return sizes, nil
}

// ListDigitalOceanImages returns images for a DigitalOcean cloud token (Coolify >= v4.2.0).
func (c *Client) ListDigitalOceanImages(ctx context.Context, cloudProviderTokenUUID string) ([]DigitalOceanImage, error) {
	var images []DigitalOceanImage
	if err := c.do(ctx, http.MethodGet, digitalOceanPath("/api/v1/digitalocean/images", cloudProviderTokenUUID), nil, &images); err != nil {
		return nil, fmt.Errorf("listing digitalocean images: %w", err)
	}
	return images, nil
}

// ListDigitalOceanSSHKeys returns SSH keys for a DigitalOcean cloud token (Coolify >= v4.2.0).
func (c *Client) ListDigitalOceanSSHKeys(ctx context.Context, cloudProviderTokenUUID string) ([]DigitalOceanSSHKey, error) {
	var keys []DigitalOceanSSHKey
	if err := c.do(ctx, http.MethodGet, digitalOceanPath("/api/v1/digitalocean/ssh-keys", cloudProviderTokenUUID), nil, &keys); err != nil {
		return nil, fmt.Errorf("listing digitalocean ssh keys: %w", err)
	}
	return keys, nil
}

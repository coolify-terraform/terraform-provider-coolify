package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// VultrRegion is a Vultr region.
type VultrRegion struct {
	ID        string `json:"id"`
	City      string `json:"city"`
	Country   string `json:"country"`
	Continent string `json:"continent"`
}

// VultrPlan is a Vultr plan.
type VultrPlan struct {
	ID          string  `json:"id"`
	VCPUCount   int64   `json:"vcpu_count"`
	RAM         int64   `json:"ram"`
	Disk        int64   `json:"disk"`
	Bandwidth   int64   `json:"bandwidth"`
	MonthlyCost float64 `json:"monthly_cost"`
	Type        string  `json:"type"`
}

// VultrOS is a Vultr operating system.
type VultrOS struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Arch   string `json:"arch"`
	Family string `json:"family"`
}

// VultrSSHKey is a Vultr SSH key.
type VultrSSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DateCreated string `json:"date_created"`
}

func vultrPath(base, cloudProviderTokenUUID string) string {
	q := url.Values{}
	q.Set("cloud_provider_token_uuid", cloudProviderTokenUUID)
	return base + "?" + q.Encode()
}

// ListVultrRegions returns regions for a Vultr cloud token (Coolify >= v4.2.0).
func (c *Client) ListVultrRegions(ctx context.Context, cloudProviderTokenUUID string) ([]VultrRegion, error) {
	var regions []VultrRegion
	if err := c.do(ctx, http.MethodGet, vultrPath("/api/v1/vultr/regions", cloudProviderTokenUUID), nil, &regions); err != nil {
		return nil, fmt.Errorf("listing vultr regions: %w", err)
	}
	return regions, nil
}

// ListVultrPlans returns plans for a Vultr cloud token (Coolify >= v4.2.0).
func (c *Client) ListVultrPlans(ctx context.Context, cloudProviderTokenUUID string) ([]VultrPlan, error) {
	var plans []VultrPlan
	if err := c.do(ctx, http.MethodGet, vultrPath("/api/v1/vultr/plans", cloudProviderTokenUUID), nil, &plans); err != nil {
		return nil, fmt.Errorf("listing vultr plans: %w", err)
	}
	return plans, nil
}

// ListVultrOS returns operating systems for a Vultr cloud token (Coolify >= v4.2.0).
func (c *Client) ListVultrOS(ctx context.Context, cloudProviderTokenUUID string) ([]VultrOS, error) {
	var oses []VultrOS
	if err := c.do(ctx, http.MethodGet, vultrPath("/api/v1/vultr/os", cloudProviderTokenUUID), nil, &oses); err != nil {
		return nil, fmt.Errorf("listing vultr os: %w", err)
	}
	return oses, nil
}

// ListVultrSSHKeys returns SSH keys for a Vultr cloud token (Coolify >= v4.2.0).
func (c *Client) ListVultrSSHKeys(ctx context.Context, cloudProviderTokenUUID string) ([]VultrSSHKey, error) {
	var keys []VultrSSHKey
	if err := c.do(ctx, http.MethodGet, vultrPath("/api/v1/vultr/ssh-keys", cloudProviderTokenUUID), nil, &keys); err != nil {
		return nil, fmt.Errorf("listing vultr ssh keys: %w", err)
	}
	return keys, nil
}

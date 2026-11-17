package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ServerImageScanning is GET/PATCH /servers/{uuid}/image-scanning.
// Coolify >= v4.7.0. Scanner limit fields are present on >= v4.7.3.
// Pointers stay nil when an older response omits the key.
type ServerImageScanning struct {
	IsImageScanEnabled    bool     `json:"is_image_scan_enabled"`
	ImageScanMemoryMB     *int64   `json:"image_scan_memory_mb,omitempty"`
	ImageScanSyftTimeout  *int64   `json:"image_scan_syft_timeout,omitempty"`
	ImageScanGrypeTimeout *int64   `json:"image_scan_grype_timeout,omitempty"`
	ImageScanCPUs         *float64 `json:"image_scan_cpus,omitempty"`
	ScannerState          string   `json:"scanner_state,omitempty"`
	ScannerFailedStep     string   `json:"scanner_failed_step,omitempty"`
	ScannerBuiltAt        string   `json:"scanner_built_at,omitempty"`
	ScannerDiskUsedBytes  *int64   `json:"scanner_disk_used_bytes,omitempty"`
}

// UpdateServerImageScanningInput is the PATCH body. Send at least one field.
// A nil pointer is omitted. Limits are rejected before v4.7.3.
type UpdateServerImageScanningInput struct {
	IsImageScanEnabled    *bool    `json:"is_image_scan_enabled,omitempty"`
	ImageScanMemoryMB     *int64   `json:"image_scan_memory_mb,omitempty"`
	ImageScanSyftTimeout  *int64   `json:"image_scan_syft_timeout,omitempty"`
	ImageScanGrypeTimeout *int64   `json:"image_scan_grype_timeout,omitempty"`
	ImageScanCPUs         *float64 `json:"image_scan_cpus,omitempty"`
}

// GetServerImageScanning returns image scanning settings for one server.
func (c *Client) GetServerImageScanning(ctx context.Context, uuid string) (*ServerImageScanning, error) {
	var r ServerImageScanning
	path := fmt.Sprintf("/api/v1/servers/%s/image-scanning", url.PathEscape(uuid))
	if err := c.do(ctx, http.MethodGet, path, nil, &r); err != nil {
		return nil, fmt.Errorf("getting image scanning for server %s: %w", uuid, err)
	}
	return &r, nil
}

// UpdateServerImageScanning turns image scanning on or off for one server.
func (c *Client) UpdateServerImageScanning(ctx context.Context, uuid string, input UpdateServerImageScanningInput) (*ServerImageScanning, error) {
	var r ServerImageScanning
	path := fmt.Sprintf("/api/v1/servers/%s/image-scanning", url.PathEscape(uuid))
	if err := c.do(ctx, http.MethodPatch, path, input, &r); err != nil {
		return nil, fmt.Errorf("updating image scanning for server %s: %w", uuid, err)
	}
	return &r, nil
}

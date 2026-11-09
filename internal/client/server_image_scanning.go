package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ServerImageScanning is GET/PATCH /servers/{uuid}/image-scanning.
// Coolify >= v4.7.0. The only writable field is IsImageScanEnabled.
type ServerImageScanning struct {
	IsImageScanEnabled   bool   `json:"is_image_scan_enabled"`
	ScannerState         string `json:"scanner_state,omitempty"`
	ScannerFailedStep    string `json:"scanner_failed_step,omitempty"`
	ScannerBuiltAt       string `json:"scanner_built_at,omitempty"`
	ScannerDiskUsedBytes *int64 `json:"scanner_disk_used_bytes,omitempty"`
}

// UpdateServerImageScanningInput is the PATCH body. The bool is required
// when the request is sent. A nil pointer means the caller should skip PATCH.
type UpdateServerImageScanningInput struct {
	IsImageScanEnabled *bool `json:"is_image_scan_enabled,omitempty"`
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

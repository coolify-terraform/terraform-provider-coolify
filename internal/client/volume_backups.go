package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// VolumeBackupSchedule is a Coolify scheduled backup for a persistent volume
// or directory storage. API requires Coolify builds that include
// VolumeBackupsController (tip / nightly after v4.2.0).
type VolumeBackupSchedule struct {
	UUID                     string  `json:"uuid"`
	Message                  string  `json:"message,omitempty"`
	StorageUUID              string  `json:"storage_uuid"`
	StorageType              string  `json:"storage_type"` // persistent | directory
	Frequency                string  `json:"frequency"`
	Enabled                  bool    `json:"enabled"`
	SaveS3                   bool    `json:"save_s3"`
	DisableLocalBackup       bool    `json:"disable_local_backup"`
	StopDuringBackup         bool    `json:"stop_during_backup"`
	S3StorageUUID            string  `json:"s3_storage_uuid,omitempty"`
	RetentionAmountLocally   int64   `json:"retention_amount_locally"`
	RetentionDaysLocally     int64   `json:"retention_days_locally"`
	RetentionMaxStorageLocal float64 `json:"retention_max_storage_locally"`
	RetentionAmountS3        int64   `json:"retention_amount_s3"`
	RetentionDaysS3          int64   `json:"retention_days_s3"`
	RetentionMaxStorageS3    float64 `json:"retention_max_storage_s3"`
	Timeout                  int64   `json:"timeout"`
}

// UpsertVolumeBackupInput is the body for PUT .../storages/{storage_uuid}/backups.
type UpsertVolumeBackupInput struct {
	Frequency                string   `json:"frequency"`
	Enabled                  *bool    `json:"enabled,omitempty"`
	SaveS3                   *bool    `json:"save_s3,omitempty"`
	DisableLocalBackup       *bool    `json:"disable_local_backup,omitempty"`
	StopDuringBackup         *bool    `json:"stop_during_backup,omitempty"`
	S3StorageUUID            string   `json:"s3_storage_uuid,omitempty"`
	RetentionAmountLocally   *int64   `json:"retention_amount_locally,omitempty"`
	RetentionDaysLocally     *int64   `json:"retention_days_locally,omitempty"`
	RetentionMaxStorageLocal *float64 `json:"retention_max_storage_locally,omitempty"`
	RetentionAmountS3        *int64   `json:"retention_amount_s3,omitempty"`
	RetentionDaysS3          *int64   `json:"retention_days_s3,omitempty"`
	RetentionMaxStorageS3    *float64 `json:"retention_max_storage_s3,omitempty"`
	Timeout                  *int64   `json:"timeout,omitempty"`
}

// UpsertVolumeBackup creates or replaces the backup schedule for a storage volume.
// parentType must be applications, databases, or services.
// Coolify returns 201 on create and 200 on replace; both are accepted.
func (c *Client) UpsertVolumeBackup(ctx context.Context, parentType, parentUUID, storageUUID string, input UpsertVolumeBackupInput) (*VolumeBackupSchedule, error) {
	if err := validateParentType(parentType); err != nil {
		return nil, fmt.Errorf("setting volume backup for %s %s storage %s: %w", parentType, parentUUID, storageUUID, err)
	}
	var r VolumeBackupSchedule
	path := fmt.Sprintf("/api/v1/%s/%s/storages/%s/backups",
		parentType, url.PathEscape(parentUUID), url.PathEscape(storageUUID))
	if err := c.do(ctx, http.MethodPut, path, input, &r); err != nil {
		return nil, fmt.Errorf("setting volume backup for %s %s storage %s: %w", parentType, parentUUID, storageUUID, err)
	}
	if r.UUID == "" {
		return nil, fmt.Errorf("setting volume backup for %s %s storage %s: API returned empty UUID", parentType, parentUUID, storageUUID)
	}
	return &r, nil
}

// DeleteVolumeBackup deletes the backup schedule (and archives) for a storage volume.
func (c *Client) DeleteVolumeBackup(ctx context.Context, parentType, parentUUID, storageUUID string) error {
	if err := validateParentType(parentType); err != nil {
		return fmt.Errorf("deleting volume backup for %s %s storage %s: %w", parentType, parentUUID, storageUUID, err)
	}
	path := fmt.Sprintf("/api/v1/%s/%s/storages/%s/backups",
		parentType, url.PathEscape(parentUUID), url.PathEscape(storageUUID))
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("deleting volume backup for %s %s storage %s: %w", parentType, parentUUID, storageUUID, err)
	}
	return nil
}

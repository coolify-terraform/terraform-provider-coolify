package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// S3Storage is an S3-compatible storage configuration in Coolify.
// Key and Secret are sensitive; Coolify omits them unless the API token
// has can_read_sensitive permission.
type S3Storage struct {
	UUID                 string `json:"uuid"`
	Name                 string `json:"name"`
	Description          string `json:"description,omitempty"`
	Endpoint             string `json:"endpoint"`
	Bucket               string `json:"bucket"`
	Region               string `json:"region"`
	Key                  string `json:"key,omitempty"`
	Secret               string `json:"secret,omitempty"`
	UsePathStyleEndpoint *bool  `json:"use_path_style_endpoint,omitempty"`
	IsUsable             *bool  `json:"is_usable,omitempty"`
	Type                 string `json:"type,omitempty"`
	SkipTLSVerification  *bool  `json:"skip_tls_verification,omitempty"`
	PrivateKeyUUID       string `json:"private_key_uuid,omitempty"`
	SFTPHostKey          string `json:"sftp_host_key,omitempty"`
}

// CreateS3StorageInput is the payload for creating S3 storage.
// Required: name, endpoint, bucket, region, key, secret.
type CreateS3StorageInput struct {
	Name                 string `json:"name"`
	Description          string `json:"description,omitempty"`
	Endpoint             string `json:"endpoint"`
	Bucket               string `json:"bucket,omitempty"`
	Region               string `json:"region,omitempty"`
	Key                  string `json:"key"`
	Secret               string `json:"secret,omitempty"`
	UsePathStyleEndpoint *bool  `json:"use_path_style_endpoint,omitempty"`
	IsUsable             *bool  `json:"is_usable,omitempty"`
	Type                 string `json:"type,omitempty"`
	SkipTLSVerification  *bool  `json:"skip_tls_verification,omitempty"`
	PrivateKeyUUID       string `json:"private_key_uuid,omitempty"`
}

// UpdateS3StorageInput is the payload for updating S3 storage.
// All fields are optional; only non-nil fields are sent.
type UpdateS3StorageInput struct {
	Name                 *string `json:"name,omitempty"`
	Description          *string `json:"description,omitempty"`
	Endpoint             *string `json:"endpoint,omitempty"`
	Bucket               *string `json:"bucket,omitempty"`
	Region               *string `json:"region,omitempty"`
	Key                  *string `json:"key,omitempty"`
	Secret               *string `json:"secret,omitempty"`
	UsePathStyleEndpoint *bool   `json:"use_path_style_endpoint,omitempty"`
	IsUsable             *bool   `json:"is_usable,omitempty"`
	Type                 *string `json:"type,omitempty"`
	SkipTLSVerification  *bool   `json:"skip_tls_verification,omitempty"`
	PrivateKeyUUID       *string `json:"private_key_uuid,omitempty"`
	// ClearPrivateKey sends private_key_uuid as JSON null. A nil pointer is
	// omitted, and Coolify keeps the key when the field is absent.
	ClearPrivateKey bool `json:"-"`
}

// ListS3Storages returns all S3 storages for the authenticated team.
// Requires Coolify >= v4.3.0.
func (c *Client) ListS3Storages(ctx context.Context) ([]S3Storage, error) {
	var r []S3Storage
	if err := c.do(ctx, http.MethodGet, "/api/v1/s3-storages", nil, &r); err != nil {
		return nil, fmt.Errorf("listing s3 storages: %w", err)
	}
	return r, nil
}

// GetS3Storage returns a single S3 storage by UUID.
// Requires Coolify >= v4.3.0.
func (c *Client) GetS3Storage(ctx context.Context, uuid string) (*S3Storage, error) {
	var r S3Storage
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/s3-storages/%s", url.PathEscape(uuid)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting s3 storage %s: %w", uuid, err)
	}
	return &r, nil
}

// CreateS3Storage creates a new S3 storage configuration.
// Coolify returns 201 with {uuid: "..."}. Requires Coolify >= v4.3.0.
func (c *Client) CreateS3Storage(ctx context.Context, input CreateS3StorageInput) (*S3Storage, error) {
	if c != nil && !c.SupportsCoolify451() {
		input.UsePathStyleEndpoint = nil
	}
	if c != nil && !c.SupportsCoolify47() {
		input.Type = ""
		input.SkipTLSVerification = nil
		input.PrivateKeyUUID = ""
	} else {
		input.restrictByType()
	}
	var r S3Storage
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/s3-storages", input, &r, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating s3 storage: %w", err)
	}
	return &r, nil
}

// UpdateS3Storage updates an existing S3 storage by UUID.
// Requires Coolify >= v4.3.0.
func (c *Client) UpdateS3Storage(ctx context.Context, uuid string, input UpdateS3StorageInput) (*S3Storage, error) {
	if c != nil && !c.SupportsCoolify451() {
		input.UsePathStyleEndpoint = nil
	}
	if c != nil && !c.SupportsCoolify47() {
		input.Type = nil
		input.SkipTLSVerification = nil
		input.PrivateKeyUUID = nil
	} else if input.Type != nil {
		input.restrictByType()
	}
	// Coolify's update allow list does not include type. Sending it is a 422.
	input.Type = nil
	body, err := storageUpdateBody(input)
	if err != nil {
		return nil, fmt.Errorf("updating s3 storage %s: %w", uuid, err)
	}
	if jsonObjectEmpty(body) {
		ver := ""
		if c != nil {
			ver = c.CoolifyVersion
		}
		tflog.Debug(ctx, "skipping s3 storage update: version gate left an empty body", map[string]interface{}{
			"uuid":    uuid,
			"version": ver,
		})
		return &S3Storage{UUID: uuid}, nil
	}
	var r S3Storage
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/s3-storages/%s", url.PathEscape(uuid)), body, &r); err != nil {
		return nil, fmt.Errorf("updating s3 storage %s: %w", uuid, err)
	}
	return &r, nil
}

func storageUpdateBody(input UpdateS3StorageInput) (any, error) {
	if !input.ClearPrivateKey {
		return input, nil
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	body["private_key_uuid"] = nil
	return body, nil
}

// DeleteS3Storage deletes an S3 storage by UUID.
// Requires Coolify >= v4.3.0.
func (c *Client) DeleteS3Storage(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/s3-storages/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting s3 storage %s: %w", uuid, err)
	}
	return nil
}

// nonS3BackupType reports Coolify backup types whose create and update
// bodies must not include S3 bucket, region, path-style, or skip-tls.
func nonS3BackupType(t string) bool {
	switch t {
	case "sftp", "ftps", "webdav":
		return true
	default:
		return false
	}
}

func (in *CreateS3StorageInput) restrictByType() {
	if nonS3BackupType(in.Type) {
		in.Bucket = ""
		in.Region = ""
		in.UsePathStyleEndpoint = nil
		in.SkipTLSVerification = nil
		return
	}
	in.PrivateKeyUUID = ""
}

func (in *UpdateS3StorageInput) restrictByType() {
	if in.Type == nil {
		return
	}
	in.omitDisallowed(*in.Type)
}

// OmitDisallowedStorageUpdate drops JSON keys Coolify rejects for this
// storage type. An empty type is s3. Call this with the planned type even
// when the PATCH does not include type.
func OmitDisallowedStorageUpdate(in *UpdateS3StorageInput, storageType string) {
	if in == nil {
		return
	}
	in.omitDisallowed(storageType)
}

func (in *UpdateS3StorageInput) omitDisallowed(storageType string) {
	if nonS3BackupType(storageType) {
		in.Bucket = nil
		in.Region = nil
		in.UsePathStyleEndpoint = nil
		in.SkipTLSVerification = nil
		return
	}
	in.PrivateKeyUUID = nil
}

// ValidateS3Storage tests the S3 connection and updates is_usable on the server.
// Requires Coolify >= v4.3.0.
func (c *Client) ValidateS3Storage(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/s3-storages/%s/validate", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("validating s3 storage %s: %w", uuid, err)
	}
	return nil
}

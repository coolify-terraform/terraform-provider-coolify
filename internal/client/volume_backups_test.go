package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_UpsertVolumeBackup_Create(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/applications/{app}/storages/{storage}/backups", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "app-1", r.PathValue("app"))
		assert.Equal(t, "stor-1", r.PathValue("storage"))
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "0 2 * * *", body["frequency"])
		assert.Equal(t, true, body["enabled"])
		assert.Equal(t, true, body["save_s3"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(VolumeBackupSchedule{
			UUID: "vb-1", StorageUUID: "stor-1", StorageType: "persistent",
			Frequency: "0 2 * * *", Enabled: true, SaveS3: true,
			S3StorageUUID: "s3-1", RetentionAmountLocally: 7, Timeout: 3600,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	en, s3 := true, true
	got, err := c.UpsertVolumeBackup(context.Background(), "applications", "app-1", "stor-1", UpsertVolumeBackupInput{
		Frequency: "0 2 * * *", Enabled: &en, SaveS3: &s3, S3StorageUUID: "s3-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "vb-1", got.UUID)
	assert.Equal(t, "persistent", got.StorageType)
}

func TestClient_UpsertVolumeBackup_Replace200(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/databases/{db}/storages/{storage}/backups", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(VolumeBackupSchedule{
			UUID: "vb-2", StorageUUID: "stor-2", StorageType: "directory",
			Frequency: "hourly", Enabled: false, Timeout: 600,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	got, err := c.UpsertVolumeBackup(context.Background(), "databases", "db-1", "stor-2", UpsertVolumeBackupInput{
		Frequency: "hourly",
	})
	require.NoError(t, err)
	assert.Equal(t, "vb-2", got.UUID)
	assert.Equal(t, "directory", got.StorageType)
}

func TestClient_UpsertVolumeBackup_EmptyUUID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/services/{svc}/storages/{storage}/backups", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(VolumeBackupSchedule{Frequency: "daily"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.UpsertVolumeBackup(context.Background(), "services", "svc-1", "stor-3", UpsertVolumeBackupInput{Frequency: "daily"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty UUID")
}

func TestClient_DeleteVolumeBackup(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/applications/{app}/storages/{storage}/backups", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "app-1", r.PathValue("app"))
		assert.Equal(t, "stor-1", r.PathValue("storage"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	require.NoError(t, c.DeleteVolumeBackup(context.Background(), "applications", "app-1", "stor-1"))
}

func TestClient_UpsertVolumeBackup_InvalidParent(t *testing.T) {
	t.Parallel()
	c := New("http://example.invalid", "t")
	_, err := c.UpsertVolumeBackup(context.Background(), "widgets", "x", "y", UpsertVolumeBackupInput{Frequency: "daily"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parent type")
}

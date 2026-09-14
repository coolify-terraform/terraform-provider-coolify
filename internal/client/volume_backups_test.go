package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "hourly", body["frequency"])
		assert.Equal(t, false, body["enabled"])
		assert.Equal(t, true, body["save_s3"])
		assert.Equal(t, float64(5), body["retention_amount_locally"])
		assert.Equal(t, float64(600), body["timeout"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(VolumeBackupSchedule{
			UUID: "vb-2", StorageUUID: "stor-2", StorageType: "directory",
			Frequency: "hourly", Enabled: false, SaveS3: true, Timeout: 600,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	en, s3 := false, true
	retention, timeout := int64(5), int64(600)
	got, err := c.UpsertVolumeBackup(context.Background(), "databases", "db-1", "stor-2", UpsertVolumeBackupInput{
		Frequency: "hourly", Enabled: &en, SaveS3: &s3,
		RetentionAmountLocally: &retention, Timeout: &timeout,
	})
	require.NoError(t, err)
	assert.Equal(t, "vb-2", got.UUID)
	assert.Equal(t, "directory", got.StorageType)
	assert.False(t, got.Enabled)
	assert.True(t, got.SaveS3)
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

func TestClient_UpsertVolumeBackup_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/applications/{app}/storages/{storage}/backups", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "missing-app", r.PathValue("app"))
		assert.Equal(t, "missing-stor", r.PathValue("storage"))
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.UpsertVolumeBackup(context.Background(), "applications", "missing-app", "missing-stor", UpsertVolumeBackupInput{
		Frequency: "daily",
	})
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.Contains(t, err.Error(), "setting volume backup")
}

func TestClient_UpsertVolumeBackup_Unprocessable(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/applications/{app}/storages/{storage}/backups", func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"validation failed"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	_, err := c.UpsertVolumeBackup(context.Background(), "applications", "app-1", "stor-1", UpsertVolumeBackupInput{
		Frequency: "not-a-cron",
	})
	require.Error(t, err)
	assert.False(t, IsNotFound(err))
	assert.Contains(t, err.Error(), "status 422")
	assert.Contains(t, err.Error(), "setting volume backup")
	assert.Equal(t, int32(1), attempts.Load(), "PUT 422 must not be retried")
}

func TestClient_DeleteVolumeBackup_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/applications/{app}/storages/{storage}/backups", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "missing-app", r.PathValue("app"))
		assert.Equal(t, "missing-stor", r.PathValue("storage"))
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(srv.URL, "test-token")
	err := c.DeleteVolumeBackup(context.Background(), "applications", "missing-app", "missing-stor")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.Contains(t, err.Error(), "deleting volume backup")
}

package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateServer_StripsSSHHostKeyBefore47(t *testing.T) {
	t.Parallel()
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"srv-1","name":"s","ip":"10.0.0.1"}`))
	}))
	t.Cleanup(srv.Close)

	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKeyForCoolify47"
	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.6.1"
	_, err := c.CreateServer(context.Background(), CreateServerInput{
		Name: "s", IP: "10.0.0.1", Port: 22, PrivateKeyUUID: "pk-1", SSHHostKey: &key,
	})
	require.NoError(t, err)
	_, ok := got["ssh_host_key"]
	assert.False(t, ok, "4.6.1 create sent ssh_host_key: %v", got)
}

func TestCreateServer_SendsSSHHostKeyOn47(t *testing.T) {
	t.Parallel()
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"srv-1","name":"s","ip":"10.0.0.1"}`))
	}))
	t.Cleanup(srv.Close)

	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKeyForCoolify47"
	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.7.1"
	_, err := c.CreateServer(context.Background(), CreateServerInput{
		Name: "s", IP: "10.0.0.1", Port: 22, PrivateKeyUUID: "pk-1", SSHHostKey: &key,
	})
	require.NoError(t, err)
	assert.Equal(t, key, got["ssh_host_key"])
}

func TestResetServerHostKey_PostsKey(t *testing.T) {
	t.Parallel()
	var got map[string]any
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Host key reset."}`))
	}))
	t.Cleanup(srv.Close)

	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKeyForCoolify47"
	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.7.1"
	require.NoError(t, c.ResetServerHostKey(context.Background(), "srv-1", &key))
	assert.Equal(t, "/api/v1/servers/srv-1/host-key/reset", path)
	assert.Equal(t, key, got["ssh_host_key"])
}

func TestCreateScheduledTask_StripsUseShellBefore47(t *testing.T) {
	t.Parallel()
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"task-1"}`))
	}))
	t.Cleanup(srv.Close)

	shell := false
	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.6.1"
	_, err := c.CreateScheduledTask(context.Background(), "applications", "app-1", CreateScheduledTaskInput{
		Name: "cleanup", Command: "echo hi", Frequency: "0 * * * *", UseShell: &shell,
	})
	require.NoError(t, err)
	_, ok := got["use_shell"]
	assert.False(t, ok, "body: %v", got)
}

func TestCreateS3Storage_SendsTypeOn47(t *testing.T) {
	t.Parallel()
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"s3-1"}`))
	}))
	t.Cleanup(srv.Close)

	skip := true
	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.7.1"
	_, err := c.CreateS3Storage(context.Background(), CreateS3StorageInput{
		Name: "s", Endpoint: "https://s3.example.com", Bucket: "b", Region: "us-east-1",
		Key: "ak", Secret: "sk", Type: "sftp", SkipTLSVerification: &skip, PrivateKeyUUID: "pk-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "sftp", got["type"])
	assert.Equal(t, true, got["skip_tls_verification"])
	assert.Equal(t, "pk-1", got["private_key_uuid"])
}

func TestUpdateDiscord_StripsImageVulnerabilitiesBefore47(t *testing.T) {
	t.Parallel()
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"discord_enabled":true}`))
	}))
	t.Cleanup(srv.Close)

	on := true
	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.6.1"
	_, err := c.UpdateDiscordNotifications(context.Background(), UpdateDiscordNotificationInput{
		Enabled: &on, ImageVulnerabilities: &on,
	})
	require.NoError(t, err)
	_, ok := got["image_vulnerabilities_discord_notifications"]
	assert.False(t, ok, "body: %v", got)
	assert.Equal(t, true, got["discord_enabled"])
}

func TestUpdateCurrentTeam_SkipsPatchBefore47(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	patched := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			mu.Lock()
			patched = true
			mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":1,"name":"root"}`))
	}))
	t.Cleanup(srv.Close)

	on := true
	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.6.1"
	got, err := c.UpdateCurrentTeam(context.Background(), UpdateCurrentTeamInput{ScanImagesOnNewServers: &on})
	require.NoError(t, err)
	require.NotNil(t, got)
	mu.Lock()
	defer mu.Unlock()
	assert.False(t, patched, "older Coolify must not receive PATCH /team for the 4.7 field")
}

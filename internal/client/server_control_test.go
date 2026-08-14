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

const testServerUUID = "aaaa0001-0001-4000-8000-000000000001"

func TestServerProxyUpdate_JSONTags(t *testing.T) {
	t.Parallel()
	enabled := false
	url := "https://example.com"
	labels := true
	proxyType := "caddy"
	input := ServerProxyUpdateInput{
		RedirectEnabled:     &enabled,
		RedirectURL:         &url,
		GenerateExactLabels: &labels,
		ProxyType:           &proxyType,
	}
	raw, err := json.Marshal(input)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, false, got["redirect_enabled"])
	assert.Equal(t, "https://example.com", got["redirect_url"])
	assert.Equal(t, true, got["generate_exact_labels"])
	assert.Equal(t, "caddy", got["proxy_type"])

	empty, err := json.Marshal(ServerProxyUpdateInput{})
	require.NoError(t, err)
	var emptyRaw map[string]any
	require.NoError(t, json.Unmarshal(empty, &emptyRaw))
	assert.Empty(t, emptyRaw)
}

func TestClient_GetUpdateServerProxy(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/servers/"+testServerUUID+"/proxy", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(ServerProxy{ProxyType: "traefik", RedirectURL: "https://example.com"})
		case http.MethodPatch:
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "caddy", body["proxy_type"])
			assert.Equal(t, "https://example.com", body["redirect_url"])
			_ = json.NewEncoder(w).Encode(ServerProxy{ProxyType: "caddy", RedirectURL: "https://example.com"})
		default:
			http.Error(w, r.URL.Path, http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")

	got, err := c.GetServerProxy(context.Background(), testServerUUID)
	require.NoError(t, err)
	assert.Equal(t, "traefik", got.ProxyType)
	assert.Equal(t, "https://example.com", got.RedirectURL)

	url := "https://example.com"
	proxyType := "caddy"
	updated, err := c.UpdateServerProxy(context.Background(), testServerUUID, ServerProxyUpdateInput{
		RedirectURL: &url,
		ProxyType:   &proxyType,
	})
	require.NoError(t, err)
	assert.Equal(t, "caddy", updated.ProxyType)
}

func TestClient_PutServerProxyConfiguration(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/api/v1/servers/"+testServerUUID+"/proxy/configuration", r.URL.Path)
		var body ServerProxyConfigInput
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "http:\n  routers:\n    web: {}", body.Configuration)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	require.NoError(t, c.PutServerProxyConfiguration(context.Background(), testServerUUID, "http:\n  routers:\n    web: {}"))
}

func TestClient_GetUpdateServerLogDrains(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/servers/"+testServerUUID+"/log-drains", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		enabled := true
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(ServerLogDrains{IsAxiomEnabled: &enabled, AxiomDatasetName: "coolify"})
		case http.MethodPatch:
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, true, body["is_logdrain_axiom_enabled"])
			assert.Equal(t, "coolify-prod", body["logdrain_axiom_dataset_name"])
			_ = json.NewEncoder(w).Encode(ServerLogDrains{IsAxiomEnabled: &enabled, AxiomDatasetName: "coolify-prod"})
		default:
			http.Error(w, r.URL.Path, http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")

	got, err := c.GetServerLogDrains(context.Background(), testServerUUID)
	require.NoError(t, err)
	require.NotNil(t, got.IsAxiomEnabled)
	assert.True(t, *got.IsAxiomEnabled)
	assert.Equal(t, "coolify", got.AxiomDatasetName)

	enabled := true
	updated, err := c.UpdateServerLogDrains(context.Background(), testServerUUID, ServerLogDrains{
		IsAxiomEnabled:   &enabled,
		AxiomDatasetName: "coolify-prod",
	})
	require.NoError(t, err)
	assert.Equal(t, "coolify-prod", updated.AxiomDatasetName)
}

func TestClient_GetUpdateServerCloudflareTunnel(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/servers/"+testServerUUID+"/cloudflare-tunnel", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(ServerCloudflareTunnel{IsCloudflareTunnel: false})
		case http.MethodPatch:
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, true, body["is_cloudflare_tunnel"])
			_ = json.NewEncoder(w).Encode(ServerCloudflareTunnel{IsCloudflareTunnel: true})
		default:
			http.Error(w, r.URL.Path, http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")

	got, err := c.GetServerCloudflareTunnel(context.Background(), testServerUUID)
	require.NoError(t, err)
	assert.False(t, got.IsCloudflareTunnel)

	updated, err := c.UpdateServerCloudflareTunnel(context.Background(), testServerUUID, true)
	require.NoError(t, err)
	assert.True(t, updated.IsCloudflareTunnel)
}

func TestClient_GetUpdateServerSentinel(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/servers/"+testServerUUID+"/sentinel", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		enabled := true
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(ServerSentinel{IsSentinelEnabled: &enabled, SentinelToken: "tok"})
		case http.MethodPatch:
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, true, body["is_sentinel_enabled"])
			assert.Equal(t, true, body["is_metrics_enabled"])
			_ = json.NewEncoder(w).Encode(ServerSentinel{IsSentinelEnabled: &enabled, IsMetricsEnabled: &enabled})
		default:
			http.Error(w, r.URL.Path, http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")

	got, err := c.GetServerSentinel(context.Background(), testServerUUID)
	require.NoError(t, err)
	require.NotNil(t, got.IsSentinelEnabled)
	assert.True(t, *got.IsSentinelEnabled)
	assert.Equal(t, "tok", got.SentinelToken)

	enabled := true
	updated, err := c.UpdateServerSentinel(context.Background(), testServerUUID, ServerSentinel{
		IsSentinelEnabled: &enabled,
		IsMetricsEnabled:  &enabled,
	})
	require.NoError(t, err)
	require.NotNil(t, updated.IsMetricsEnabled)
	assert.True(t, *updated.IsMetricsEnabled)
}

func TestClient_GetUpdateServerDockerCleanup(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/servers/"+testServerUUID+"/docker-cleanup", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		threshold := int64(70)
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(ServerDockerCleanup{DockerCleanupFrequency: "@daily", DockerCleanupThreshold: &threshold})
		case http.MethodPatch:
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "daily", body["docker_cleanup_frequency"])
			assert.Equal(t, float64(80), body["docker_cleanup_threshold"])
			updated := int64(80)
			_ = json.NewEncoder(w).Encode(ServerDockerCleanup{DockerCleanupFrequency: "daily", DockerCleanupThreshold: &updated})
		default:
			http.Error(w, r.URL.Path, http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")

	got, err := c.GetServerDockerCleanup(context.Background(), testServerUUID)
	require.NoError(t, err)
	assert.Equal(t, "@daily", got.DockerCleanupFrequency)
	require.NotNil(t, got.DockerCleanupThreshold)
	assert.Equal(t, int64(70), *got.DockerCleanupThreshold)

	threshold := int64(80)
	updated, err := c.UpdateServerDockerCleanup(context.Background(), testServerUUID, ServerDockerCleanup{
		DockerCleanupFrequency: "daily",
		DockerCleanupThreshold: &threshold,
	})
	require.NoError(t, err)
	assert.Equal(t, "daily", updated.DockerCleanupFrequency)
	require.NotNil(t, updated.DockerCleanupThreshold)
	assert.Equal(t, int64(80), *updated.DockerCleanupThreshold)
}

func TestClient_ApplicationDestinations(t *testing.T) {
	t.Parallel()
	const destUUID = "dddd0001-0001-4000-8000-000000000001"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications/app-1/destinations":
			_ = json.NewEncoder(w).Encode([]ApplicationDestination{{UUID: destUUID, IsPrimary: true}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/applications/app-1/destinations":
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, destUUID, body["destination_uuid"])
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/applications/app-1/destinations/"+destUUID:
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, r.URL.Path, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")

	list, err := c.ListApplicationDestinations(context.Background(), "app-1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, destUUID, list[0].UUID)
	assert.True(t, list[0].IsPrimary)

	require.NoError(t, c.AttachApplicationDestination(context.Background(), "app-1", destUUID))
	require.NoError(t, c.DetachApplicationDestination(context.Background(), "app-1", destUUID))
}

func TestClient_AttachApplicationDestination_WrongStatus(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")
	err := c.AttachApplicationDestination(context.Background(), "app-1", "dest-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "attaching destination dest-1 to application app-1")
}

func TestClient_ServerControl_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Not found."}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-token")

	checks := []struct {
		name string
		fn   func() error
	}{
		{"GetServerProxy", func() error { _, err := c.GetServerProxy(context.Background(), "missing"); return err }},
		{"UpdateServerProxy", func() error {
			_, err := c.UpdateServerProxy(context.Background(), "missing", ServerProxyUpdateInput{})
			return err
		}},
		{"PutServerProxyConfiguration", func() error { return c.PutServerProxyConfiguration(context.Background(), "missing", "x") }},
		{"GetServerLogDrains", func() error { _, err := c.GetServerLogDrains(context.Background(), "missing"); return err }},
		{"UpdateServerLogDrains", func() error {
			_, err := c.UpdateServerLogDrains(context.Background(), "missing", ServerLogDrains{})
			return err
		}},
		{"GetServerCloudflareTunnel", func() error { _, err := c.GetServerCloudflareTunnel(context.Background(), "missing"); return err }},
		{"UpdateServerCloudflareTunnel", func() error {
			_, err := c.UpdateServerCloudflareTunnel(context.Background(), "missing", true)
			return err
		}},
		{"GetServerSentinel", func() error { _, err := c.GetServerSentinel(context.Background(), "missing"); return err }},
		{"UpdateServerSentinel", func() error {
			_, err := c.UpdateServerSentinel(context.Background(), "missing", ServerSentinel{})
			return err
		}},
		{"GetServerDockerCleanup", func() error { _, err := c.GetServerDockerCleanup(context.Background(), "missing"); return err }},
		{"UpdateServerDockerCleanup", func() error {
			_, err := c.UpdateServerDockerCleanup(context.Background(), "missing", ServerDockerCleanup{})
			return err
		}},
		{"ListApplicationDestinations", func() error { _, err := c.ListApplicationDestinations(context.Background(), "missing"); return err }},
		{"AttachApplicationDestination", func() error { return c.AttachApplicationDestination(context.Background(), "missing", "dest") }},
		{"DetachApplicationDestination", func() error { return c.DetachApplicationDestination(context.Background(), "missing", "dest") }},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			require.Error(t, err)
			assert.True(t, IsNotFound(err), "expected NotFound, got %v", err)
		})
	}
}

package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetCloudInitScript_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetCloudInitScript(context.Background(), "nonexistent-uuid")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_CreateCloudInitScript(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/cloud-init-scripts", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CloudInitScriptInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "bootstrap", input.Name)
		assert.Equal(t, "#cloud-config\npackages: [nginx]\n", input.Script)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CloudInitScript{
			UUID:   "ci-new",
			Name:   "bootstrap",
			Script: "#cloud-config\npackages: [nginx]\n",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.CreateCloudInitScript(context.Background(), CloudInitScriptInput{
		Name:   "bootstrap",
		Script: "#cloud-config\npackages: [nginx]\n",
	})
	require.NoError(t, err)
	assert.Equal(t, "ci-new", got.UUID)
	assert.Equal(t, "bootstrap", got.Name)
	assert.Equal(t, "#cloud-config\npackages: [nginx]\n", got.Script)
}

func TestClient_CreateCloudInitScript_WrongStatusCode(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200, not the expected 201
		_ = json.NewEncoder(w).Encode(CloudInitScript{UUID: "ci-wrong"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.CreateCloudInitScript(context.Background(), CloudInitScriptInput{
		Name: "t", Script: "#cloud-config\n",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected status 201")
	assert.Contains(t, err.Error(), "got 200")
}

func TestClient_UpdateCloudInitScript_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/cloud-init-scripts/missing", r.URL.Path)
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.UpdateCloudInitScript(context.Background(), "missing", CloudInitScriptInput{
		Name: "renamed",
	})
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_DeleteCloudInitScript_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/cloud-init-scripts/missing", r.URL.Path)
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteCloudInitScript(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClient_ListCloudInitScripts(t *testing.T) {
	t.Parallel()
	want := []CloudInitScript{
		{UUID: "ci-1", Name: "bootstrap", Script: "#cloud-config\npackages: [nginx]\n"},
		{UUID: "ci-2", Name: "docker", Script: "#cloud-config\nruncmd:\n  - docker\n"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/cloud-init-scripts", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	got, err := c.ListCloudInitScripts(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestClient_ListCloudInitScripts_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.ListCloudInitScripts(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing cloud-init scripts")
}

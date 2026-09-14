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

func TestClient_GetGitLabApp_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]GitLabApp{
			{ID: 99, UUID: "gl-other", Name: "other-app"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetGitLabApp(context.Background(), 42)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.Contains(t, err.Error(), "42")
}

func TestClient_GetGitLabAppByUUID_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]GitLabApp{
			{ID: 1, UUID: "gl-1", Name: "other-app"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.GetGitLabAppByUUID(context.Background(), "missing-uuid")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.Contains(t, err.Error(), "missing-uuid")
}

func TestClient_CreateGitLabApp(t *testing.T) {
	t.Parallel()
	port := int64(2222)
	sysWide := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/gitlab-apps", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input CreateGitLabAppInput
		require.NoError(t, json.Unmarshal(body, &input))
		assert.Equal(t, "My GitLab App", input.Name)
		assert.Equal(t, "https://gitlab.example.com", input.HTMLURL)
		assert.Equal(t, "https://gitlab.example.com/api/v4", input.APIURL)
		assert.Equal(t, "git", input.CustomUser)
		require.NotNil(t, input.CustomPort)
		assert.Equal(t, int64(2222), *input.CustomPort)
		assert.Equal(t, "acme", input.GroupName)
		assert.Equal(t, "gl-app-id", input.ClientID)
		assert.Equal(t, "gl-secret", input.ClientSecret)
		assert.Equal(t, "gl-webhook", input.WebhookToken)
		assert.Equal(t, "https://coolify.example.com/redirect", input.RedirectURI)
		require.NotNil(t, input.IsSystemWide)
		assert.True(t, *input.IsSystemWide)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "GitLab app created",
			"data": GitLabApp{
				ID: 42, UUID: "gl-new", Name: "My GitLab App",
				HTMLURL: "https://gitlab.example.com",
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	app, err := c.CreateGitLabApp(context.Background(), CreateGitLabAppInput{
		Name:         "My GitLab App",
		HTMLURL:      "https://gitlab.example.com",
		APIURL:       "https://gitlab.example.com/api/v4",
		CustomUser:   "git",
		CustomPort:   &port,
		GroupName:    "acme",
		ClientID:     "gl-app-id",
		ClientSecret: "gl-secret",
		WebhookToken: "gl-webhook",
		RedirectURI:  "https://coolify.example.com/redirect",
		IsSystemWide: &sysWide,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), app.ID)
	assert.Equal(t, "gl-new", app.UUID)
	assert.Equal(t, "My GitLab App", app.Name)
	assert.Equal(t, "https://gitlab.example.com", app.HTMLURL)
}

func TestClient_CreateGitLabApp_WrongStatusCode(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200, not the expected 201
		_ = json.NewEncoder(w).Encode(GitLabApp{ID: 1, Name: "t"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	_, err := c.CreateGitLabApp(context.Background(), CreateGitLabAppInput{
		Name: "t", HTMLURL: "https://gitlab.example.com",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected status 201")
	assert.Contains(t, err.Error(), "got 200")
}

func TestClient_DeleteGitLabApp_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/gitlab-apps/999", r.URL.Path)
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	err := c.DeleteGitLabApp(context.Background(), 999)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestDecodeGitLabApp_Envelope(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"message":"ok","data":{"id":7,"uuid":"abc","name":"My App","html_url":"https://gitlab.example.com"}}`)
	app, err := decodeGitLabApp(raw)
	require.NoError(t, err)
	assert.Equal(t, int64(7), app.ID)
	assert.Equal(t, "abc", app.UUID)
	assert.Equal(t, "My App", app.Name)
	assert.Equal(t, "https://gitlab.example.com", app.HTMLURL)
}

func TestDecodeGitLabApp_Direct(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"id":9,"uuid":"def","name":"Direct App","html_url":"https://gitlab.example.com"}`)
	app, err := decodeGitLabApp(raw)
	require.NoError(t, err)
	assert.Equal(t, int64(9), app.ID)
	assert.Equal(t, "def", app.UUID)
	assert.Equal(t, "Direct App", app.Name)
	assert.Equal(t, "https://gitlab.example.com", app.HTMLURL)
}

func TestDecodeGitLabApp_InvalidJSON(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`not json`)
	_, err := decodeGitLabApp(raw)
	require.Error(t, err)
}

func TestDecodeGitLabApp_MissingID(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"name":"no-id"}`)
	_, err := decodeGitLabApp(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing id")
}

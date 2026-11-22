package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGiteaAppJSONDropsSecrets(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"id":7,
		"uuid":"550e8400-e29b-41d4-a716-446655440000",
		"name":"gitea",
		"client_secret":"sekret",
		"access_token":"tok",
		"refresh_token":"ref",
		"webhook_secret":"wh",
		"private_key_id":4,
		"is_connected":true
	}`)
	var app GiteaApp
	if err := json.Unmarshal(raw, &app); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(app)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, secret := range []string{"sekret", "tok", "ref", "wh", "client_secret", "access_token", "webhook_secret", "private_key_id"} {
		if strings.Contains(text, secret) {
			t.Fatalf("encoded app contains %s: %s", secret, text)
		}
	}
	if !app.IsConnected || app.UUID == "" {
		t.Fatalf("%+v", app)
	}
}

func TestClient_ListGiteaAppsRequires47(t *testing.T) {
	t.Parallel()
	c := New("http://127.0.0.1:1", "test-token")
	c.CoolifyVersion = "4.7.0-beta.1"
	_, err := c.ListGiteaApps(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Coolify >= v4.7.0") {
		t.Fatalf("got %v", err)
	}
}

func TestClient_ListGiteaApps(t *testing.T) {
	t.Parallel()
	body := `[{
		"id":7,
		"uuid":"550e8400-e29b-41d4-a716-446655440000",
		"name":"work",
		"organization":null,
		"api_url":"https://gitea.example/api/v1",
		"html_url":"https://gitea.example",
		"custom_user":"git",
		"custom_port":2222,
		"client_id":"client-1",
		"redirect_uri":"https://coolify.example/callback",
		"is_system_wide":true,
		"is_public":false,
		"is_connected":true,
		"team_id":3,
		"client_secret":"sekret",
		"access_token":"tok",
		"refresh_token":"ref",
		"webhook_secret":"wh",
		"created_at":"2026-11-01T00:00:00Z",
		"updated_at":"2026-11-02T00:00:00Z"
	}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/gitea-apps" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.7.6"
	apps, err := c.ListGiteaApps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].UUID == "" || apps[0].Name != "work" || !apps[0].IsConnected || !apps[0].IsSystemWide {
		t.Fatalf("%+v", apps)
	}
	if apps[0].Organization != nil || apps[0].CustomPort == nil || *apps[0].CustomPort != 2222 {
		t.Fatalf("optional fields: %+v", apps[0])
	}
	if apps[0].TeamID == nil || *apps[0].TeamID != 3 || apps[0].ClientID == nil || *apps[0].ClientID != "client-1" {
		t.Fatalf("ids: %+v", apps[0])
	}
}

func TestClient_ListGiteaAppsEmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"uuid":"","name":"broken","is_connected":false}]`))
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "test-token")
	_, err := c.ListGiteaApps(context.Background())
	if err == nil || !strings.Contains(err.Error(), "empty uuid") {
		t.Fatalf("got %v", err)
	}
}

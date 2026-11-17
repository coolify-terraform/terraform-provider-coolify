package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateServer_PublicIPsFollowCoolifyVersion(t *testing.T) {
	t.Parallel()
	var body map[string]any
	var sawPatch bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			sawPatch = true
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
			}
			body = map[string]any{}
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Errorf("decode body: %v", err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uuid":"s1"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.7.5"
	if _, err := c.UpdateServer(context.Background(), "s1", UpdateServerInput{PublicIPv4: NewJSONString("203.0.113.10")}); err != nil {
		t.Fatalf("4.7.5 update: %v", err)
	}
	if sawPatch {
		t.Fatalf("4.7.5 sent a PATCH: %v", body)
	}

	c.CoolifyVersion = "4.8.0-alpha.0.20261116.1"
	if _, err := c.UpdateServer(context.Background(), "s1", UpdateServerInput{PublicIPv4: NewJSONString("203.0.113.10")}); err != nil {
		t.Fatalf("nightly update: %v", err)
	}
	if body["public_ipv4"] != "203.0.113.10" {
		t.Fatalf("nightly PATCH = %v", body)
	}

	sawPatch = false
	body = nil
	if _, err := c.UpdateServer(context.Background(), "s1", UpdateServerInput{PublicIPv4: NewJSONNull()}); err != nil {
		t.Fatalf("clear update: %v", err)
	}
	if !sawPatch {
		t.Fatal("clear did not PATCH")
	}
	if _, ok := body["public_ipv4"]; !ok || body["public_ipv4"] != nil {
		t.Fatalf("clear PATCH = %v, want public_ipv4 null", body)
	}
}

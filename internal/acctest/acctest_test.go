package acctest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAccTestServerUUID_UsesVisibleOverride(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": "v4.1.0-test"})
	})
	mux.HandleFunc("GET /api/v1/servers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{{"uuid": "srv-visible"}, {"uuid": "srv-other"}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, kv := range [][2]string{{"COOLIFY_ENDPOINT", srv.URL}, {"COOLIFY_TOKEN", "test-token"}, {"COOLIFY_SERVER_UUID", "srv-visible"}} {
		if err := os.Setenv(kv[0], kv[1]); err != nil {
			t.Fatalf("setting %s: %v", kv[0], err)
		}
		defer os.Unsetenv(kv[0])
	}

	if got := AccTestServerUUID(t); got != "srv-visible" {
		t.Fatalf("AccTestServerUUID() = %q, want %q", got, "srv-visible")
	}
}

func TestAccTestServerUUID_FallsBackToFirstVisibleServer(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": "v4.1.0-test"})
	})
	mux.HandleFunc("GET /api/v1/servers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{{"uuid": "srv-first"}, {"uuid": "srv-second"}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, kv := range [][2]string{{"COOLIFY_ENDPOINT", srv.URL}, {"COOLIFY_TOKEN", "test-token"}} {
		if err := os.Setenv(kv[0], kv[1]); err != nil {
			t.Fatalf("setting %s: %v", kv[0], err)
		}
		defer os.Unsetenv(kv[0])
	}
	_ = os.Unsetenv("COOLIFY_SERVER_UUID")

	if got := AccTestServerUUID(t); got != "srv-first" {
		t.Fatalf("AccTestServerUUID() = %q, want %q", got, "srv-first")
	}
}

func TestAccTestServerUUID_SkipsWhenOverrideIsNotVisible(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": "v4.1.0-test"})
	})
	mux.HandleFunc("GET /api/v1/servers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{{"uuid": "srv-visible"}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, kv := range [][2]string{{"COOLIFY_ENDPOINT", srv.URL}, {"COOLIFY_TOKEN", "test-token"}, {"COOLIFY_SERVER_UUID", "srv-missing"}} {
		if err := os.Setenv(kv[0], kv[1]); err != nil {
			t.Fatalf("setting %s: %v", kv[0], err)
		}
		defer os.Unsetenv(kv[0])
	}

	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestServerUUID(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestServerUUID to skip when COOLIFY_SERVER_UUID is not visible")
	}
}

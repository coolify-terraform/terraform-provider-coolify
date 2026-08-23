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

func TestAccTestSkipIfNoVolumeBackupAPI_Present(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.2.0-edge"})
	})
	// Controller present: resource not found for fake UUIDs
	mux.HandleFunc("PUT /api/v1/applications/{app}/storages/{stor}/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Application not found."}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, kv := range [][2]string{{"COOLIFY_ENDPOINT", srv.URL}, {"COOLIFY_TOKEN", "test-token"}} {
		if err := os.Setenv(kv[0], kv[1]); err != nil {
			t.Fatalf("setting %s: %v", kv[0], err)
		}
		defer os.Unsetenv(kv[0]) //nolint:errcheck
	}

	// Must not skip when controller is present
	AccTestSkipIfNoVolumeBackupAPI(t)
}

func TestAccTestSkipIfNoVolumeBackupAPI_MissingRoute(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.2.0"})
	})
	// Unmatched PUT: Go ServeMux returns 404 with empty body (treated as missing route)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, kv := range [][2]string{{"COOLIFY_ENDPOINT", srv.URL}, {"COOLIFY_TOKEN", "test-token"}} {
		if err := os.Setenv(kv[0], kv[1]); err != nil {
			t.Fatalf("setting %s: %v", kv[0], err)
		}
		defer os.Unsetenv(kv[0]) //nolint:errcheck
	}

	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfNoVolumeBackupAPI(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfNoVolumeBackupAPI to skip when volume backup route is missing")
	}
}

func TestRequireTipAPIs(t *testing.T) {
	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")
	if requireTipAPIs() {
		t.Fatal("expected requireTipAPIs false when unset/empty")
	}
	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "1")
	if !requireTipAPIs() {
		t.Fatal("expected requireTipAPIs true when COOLIFY_REQUIRE_TIP_APIS=1")
	}
	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "true")
	if !requireTipAPIs() {
		t.Fatal("expected requireTipAPIs true when COOLIFY_REQUIRE_TIP_APIS=true")
	}
}

func TestAccTestSkipIfNoVolumeBackupAPI_ValidationPresent(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.2.0-edge"})
	})
	mux.HandleFunc("PUT /api/v1/applications/{app}/storages/{stor}/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation failed."}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, kv := range [][2]string{{"COOLIFY_ENDPOINT", srv.URL}, {"COOLIFY_TOKEN", "test-token"}} {
		if err := os.Setenv(kv[0], kv[1]); err != nil {
			t.Fatalf("setting %s: %v", kv[0], err)
		}
		defer os.Unsetenv(kv[0]) //nolint:errcheck
	}

	AccTestSkipIfNoVolumeBackupAPI(t)
}

// Coolify unmatched routes return {"message":"Not found."} (no resource name).
// That must skip, not treat the volume-backup controller as present (floor 4.1.2).
func TestAccTestSkipIfNoVolumeBackupAPI_PlainNotFound(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()
	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.1.2"})
	})
	mux.HandleFunc("PUT /api/v1/applications/{app}/storages/{stor}/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not found."}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfNoVolumeBackupAPI(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfNoVolumeBackupAPI to skip on plain Not found. body")
	}
}

func TestAccTestSkipIfNoS3StorageAPI_PlainNotFound(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()
	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.1.2"})
	})
	mux.HandleFunc("GET /api/v1/s3-storages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not found."}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfNoS3StorageAPI(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfNoS3StorageAPI to skip on plain Not found. body")
	}
}

func TestIsControllerResourceNotFound(t *testing.T) {
	if !isControllerResourceNotFound("application not found.") {
		t.Fatal("expected application not found to be controller-style")
	}
	if isControllerResourceNotFound(`{"message":"not found."}`) {
		t.Fatal("plain not found must not look like controller resource 404")
	}
	if isControllerResourceNotFound("not found.") {
		t.Fatal("plain not found. must not look like controller resource 404")
	}
}

func TestIsExtraKeyNotAllowed(t *testing.T) {
	if !isExtraKeyNotAllowed(`{"message":"validation failed.","errors":{"smtp_ehlo_domain":["this field is not allowed."]}}`) {
		t.Fatal("expected extra-key 422 body to match")
	}
	if isExtraKeyNotAllowed(`{"message":"the smtp ehlo domain field must be a valid hostname."}`) {
		t.Fatal("hostname validation 422 must not look like extra-key")
	}
}

func TestAccTestSkipIfNoSMTPEhloDomain_ExtraKey(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()
	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.0"})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation failed.","errors":{"smtp_ehlo_domain":["This field is not allowed."]}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfNoSMTPEhloDomain(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfNoSMTPEhloDomain to skip on extra-key 422")
	}

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "1")
	reachedTip := false
	t.Run("skip-under-require-tip", func(t *testing.T) {
		AccTestSkipIfNoSMTPEhloDomain(t)
		reachedTip = true
	})
	if reachedTip {
		t.Fatal("smtp_ehlo extra-key 422 must soft-skip under COOLIFY_REQUIRE_TIP_APIS=1")
	}
}

func TestAccTestSkipIfNoSMTPEhloDomain_ValidationPresent(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.0"})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode probe body: %v", err)
		}
		if body["smtp_ehlo_domain"] != smtpEhloDomainProbeValue {
			t.Fatalf("probe key = %v, want %q", body["smtp_ehlo_domain"], smtpEhloDomainProbeValue)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"The smtp ehlo domain field must be a valid hostname."}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	AccTestSkipIfNoSMTPEhloDomain(t)
}

func TestAccTestSMTPEhloDomainAccepted_OK(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.0"})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"smtp_ehlo_domain":null}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	if !AccTestSMTPEhloDomainAccepted(t) {
		t.Fatal("expected 200 PATCH to accept smtp_ehlo_domain")
	}
}

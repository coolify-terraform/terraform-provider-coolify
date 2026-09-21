package acctest

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
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
	if !isControllerResourceNotFound("preview not found.") {
		t.Fatal("expected preview not found to be controller-style")
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

func TestAccTestSkipIfNoMissingBackupNotificationDays_ExtraKey(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()
	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.0"})
	})
	mux.HandleFunc("POST /api/v1/databases/{uuid}/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation failed.","errors":{"missing_backup_notification_days":["This field is not allowed."]}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfNoMissingBackupNotificationDays(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfNoMissingBackupNotificationDays to skip on extra-key 422")
	}

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "1")
	reachedTip := false
	t.Run("skip-under-require-tip", func(t *testing.T) {
		AccTestSkipIfNoMissingBackupNotificationDays(t)
		reachedTip = true
	})
	if reachedTip {
		t.Fatal("missing_backup_notification_days extra-key 422 must soft-skip under COOLIFY_REQUIRE_TIP_APIS=1")
	}
}

func TestAccTestSkipIfNoMissingBackupNotificationDays_ValidationPresent(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.0"})
	})
	mux.HandleFunc("POST /api/v1/databases/{uuid}/backups", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode probe body: %v", err)
		}
		if body["missing_backup_notification_days"] != float64(-1) {
			t.Fatalf("probe days = %v, want -1", body["missing_backup_notification_days"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation failed.","errors":{"missing_backup_notification_days":["The missing backup notification days must be at least 0."]}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	AccTestSkipIfNoMissingBackupNotificationDays(t)
}

func TestAccTestSkipIfNoMissingBackupNotificationDays_NotFound(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()
	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.17"})
	})
	mux.HandleFunc("POST /api/v1/databases/{uuid}/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Database not found."}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfNoMissingBackupNotificationDays(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfNoMissingBackupNotificationDays to skip on 404")
	}
}

func TestAccTestSkipIfNoSMTPEhloDomain_ServerError(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.0"})
	})
	mux.HandleFunc("PATCH /api/v1/notifications/email", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal server error"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")
	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfNoSMTPEhloDomain(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfNoSMTPEhloDomain to skip on HTTP 500")
	}

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "1")
	if os.Getenv("ACC_SMTP_EHLO_PROBE_CHILD") == "1" {
		t.Run("fatal", func(t *testing.T) {
			AccTestSkipIfNoSMTPEhloDomain(t)
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestAccTestSkipIfNoSMTPEhloDomain_ServerError$")
	cmd.Env = append(os.Environ(), "ACC_SMTP_EHLO_PROBE_CHILD=1", "COOLIFY_REQUIRE_TIP_APIS=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected Fatal on HTTP 500 with COOLIFY_REQUIRE_TIP_APIS=1, child passed:\n%s", out)
	}
	if !bytes.Contains(out, []byte("COOLIFY_REQUIRE_TIP_APIS")) {
		t.Fatalf("child output missing REQUIRE_TIP fatal message:\n%s", out)
	}
}

func TestAccTestSkipIfNoMissingBackupNotificationDays_ServerError(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.0"})
	})
	mux.HandleFunc("POST /api/v1/databases/{uuid}/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal server error"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")
	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfNoMissingBackupNotificationDays(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfNoMissingBackupNotificationDays to skip on HTTP 500")
	}

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "1")
	if os.Getenv("ACC_MISSING_DAYS_PROBE_CHILD") == "1" {
		t.Run("fatal", func(t *testing.T) {
			AccTestSkipIfNoMissingBackupNotificationDays(t)
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestAccTestSkipIfNoMissingBackupNotificationDays_ServerError$")
	cmd.Env = append(os.Environ(), "ACC_MISSING_DAYS_PROBE_CHILD=1", "COOLIFY_REQUIRE_TIP_APIS=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected Fatal on HTTP 500 with COOLIFY_REQUIRE_TIP_APIS=1, child passed:\n%s", out)
	}
	if !bytes.Contains(out, []byte("COOLIFY_REQUIRE_TIP_APIS")) {
		t.Fatalf("child output missing REQUIRE_TIP fatal message:\n%s", out)
	}
}

func TestAccTestSkipIfNoPreviewDomainUpdate_PlainNotFound(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()
	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.0"})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}/previews/{pr}", func(w http.ResponseWriter, _ *http.Request) {
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
		AccTestSkipIfNoPreviewDomainUpdate(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfNoPreviewDomainUpdate to skip on plain Not found. body")
	}

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "1")
	reachedTip := false
	t.Run("skip-under-require-tip", func(t *testing.T) {
		AccTestSkipIfNoPreviewDomainUpdate(t)
		reachedTip = true
	})
	if reachedTip {
		t.Fatal("preview domain unmatched 404 must soft-skip under COOLIFY_REQUIRE_TIP_APIS=1")
	}
}

func TestAccTestSkipIfNoPreviewDomainUpdate_ControllerNotFound(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.0"})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}/previews/{pr}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Preview not found."}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	AccTestSkipIfNoPreviewDomainUpdate(t)
}

func TestAccTestSkipIfNoPreviewDomainUpdate_ServerError(t *testing.T) {
	resetAccTestCaches()
	defer resetAccTestCaches()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.0"})
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}/previews/{pr}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal server error"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")
	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfNoPreviewDomainUpdate(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfNoPreviewDomainUpdate to skip on HTTP 500")
	}

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "1")
	if os.Getenv("ACC_PREVIEW_PROBE_CHILD") == "1" {
		t.Run("fatal", func(t *testing.T) {
			AccTestSkipIfNoPreviewDomainUpdate(t)
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestAccTestSkipIfNoPreviewDomainUpdate_ServerError$")
	cmd.Env = append(os.Environ(), "ACC_PREVIEW_PROBE_CHILD=1", "COOLIFY_REQUIRE_TIP_APIS=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected Fatal on HTTP 500 with COOLIFY_REQUIRE_TIP_APIS=1, child passed:\n%s", out)
	}
	if !bytes.Contains(out, []byte("COOLIFY_REQUIRE_TIP_APIS")) {
		t.Fatalf("child output missing REQUIRE_TIP fatal message:\n%s", out)
	}
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

func TestIsClearDomainsProbeInfraError(t *testing.T) {
	if isClearDomainsProbeInfraError(nil) {
		t.Fatal("nil is not infra")
	}
	if !isClearDomainsProbeInfraError(client.ErrRetriesExhausted) {
		t.Fatal("retries exhausted is infra")
	}
	if isClearDomainsProbeInfraError(&client.NotFoundError{Message: "application not found"}) {
		t.Fatal("404 is a skip, not infra")
	}
	if isClearDomainsProbeInfraError(&client.APIStatusError{Status: http.StatusUnprocessableEntity, Message: "validation"}) {
		t.Fatal("422 is a skip, not infra")
	}
	if !isClearDomainsProbeInfraError(&client.APIStatusError{Status: http.StatusInternalServerError, Message: "boom"}) {
		t.Fatal("500 is infra")
	}
	if !isClearDomainsProbeInfraError(errors.New("executing request for POST /api/v1/projects: connection refused")) {
		t.Fatal("POST transport without APIStatusError is infra")
	}
}

func TestAccTestSkipIfCannotClearApplicationDomains_Clears(t *testing.T) {
	t.Setenv("COOLIFY_SERVER_UUID", "from-host-preflight")
	startClearDomainsProbeServer(t, clearDomainsModeClears)
	AccTestSkipIfCannotClearApplicationDomains(t)
}

func TestAccTestSkipIfCannotClearApplicationDomains_Ignores(t *testing.T) {
	startClearDomainsProbeServer(t, clearDomainsModeIgnores)
	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")

	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfCannotClearApplicationDomains(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfCannotClearApplicationDomains to skip when empty PATCH leaves fqdn")
	}

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "1")
	reachedTip := false
	t.Run("skip-under-require-tip", func(t *testing.T) {
		AccTestSkipIfCannotClearApplicationDomains(t)
		reachedTip = true
	})
	if reachedTip {
		t.Fatal("empty-domains ignore must soft-skip under COOLIFY_REQUIRE_TIP_APIS=1")
	}
}

func TestAccTestSkipIfCannotClearApplicationDomains_NoPersist(t *testing.T) {
	startClearDomainsProbeServer(t, clearDomainsModeNoPersist)

	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfCannotClearApplicationDomains(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfCannotClearApplicationDomains to skip when create does not persist fqdn")
	}
}

func TestAccTestSkipIfCannotClearApplicationDomains_ServerError(t *testing.T) {
	startClearDomainsProbeServer(t, clearDomainsModeProject500)

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "")
	reached := false
	t.Run("skip", func(t *testing.T) {
		AccTestSkipIfCannotClearApplicationDomains(t)
		reached = true
	})
	if reached {
		t.Fatal("expected AccTestSkipIfCannotClearApplicationDomains to skip on HTTP 500")
	}

	t.Setenv("COOLIFY_REQUIRE_TIP_APIS", "1")
	if os.Getenv("ACC_CLEAR_DOMAINS_PROBE_CHILD") == "1" {
		t.Run("fatal", func(t *testing.T) {
			AccTestSkipIfCannotClearApplicationDomains(t)
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestAccTestSkipIfCannotClearApplicationDomains_ServerError$")
	cmd.Env = append(os.Environ(), "ACC_CLEAR_DOMAINS_PROBE_CHILD=1", "COOLIFY_REQUIRE_TIP_APIS=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected Fatal on HTTP 500 with COOLIFY_REQUIRE_TIP_APIS=1, child passed:\n%s", out)
	}
	if !bytes.Contains(out, []byte("COOLIFY_REQUIRE_TIP_APIS")) {
		t.Fatalf("child output missing REQUIRE_TIP fatal message:\n%s", out)
	}
}

const (
	clearDomainsModeClears     = "clears"
	clearDomainsModeIgnores    = "ignores"
	clearDomainsModeNoPersist  = "nopersist"
	clearDomainsModeProject500 = "project500"
)

type clearDomainsProbeAPI struct {
	mode string
	fqdn string
}

func startClearDomainsProbeServer(t *testing.T, mode string) {
	t.Helper()
	resetAccTestCaches()
	t.Cleanup(resetAccTestCaches)

	st := &clearDomainsProbeAPI{
		mode: mode,
		fqdn: "http://tf-acc-clrdmn.example.com",
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "v4.3.23"})
	})
	mux.HandleFunc("GET /api/v1/servers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]string{{"uuid": "srv-clear-1"}})
	})
	mux.HandleFunc("POST /api/v1/projects", st.handleCreateProject)
	mux.HandleFunc("POST /api/v1/applications/dockerimage", st.handleCreateApp)
	mux.HandleFunc("GET /api/v1/applications/{uuid}", st.handleGetApp)
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", st.handlePatchApp)
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", st.handleDelete)
	mux.HandleFunc("DELETE /api/v1/projects/{uuid}", st.handleDelete)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv("COOLIFY_ENDPOINT", srv.URL)
	t.Setenv("COOLIFY_TOKEN", "test-token")
	t.Setenv("COOLIFY_SERVER_UUID", "srv-clear-1")
}

func (st *clearDomainsProbeAPI) handleCreateProject(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if st.mode == clearDomainsModeProject500 {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal server error"}`))
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"uuid": "proj-clear-1", "name": "probe"})
}

func (st *clearDomainsProbeAPI) handleCreateApp(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"uuid": "app-clear-1"})
}

func (st *clearDomainsProbeAPI) handleGetApp(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fqdn := st.fqdn
	if st.mode == clearDomainsModeNoPersist {
		fqdn = ""
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"uuid": "app-clear-1", "fqdn": fqdn})
}

func (st *clearDomainsProbeAPI) handlePatchApp(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	if st.mode == clearDomainsModeClears {
		if d, ok := body["domains"].(string); ok && d == "" {
			st.fqdn = ""
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"uuid": "app-clear-1", "fqdn": st.fqdn})
}

func (st *clearDomainsProbeAPI) handleDelete(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

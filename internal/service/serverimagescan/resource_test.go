package serverimagescan_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestServerImageScanningResource_CreateSendsEnabled(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var patched map[string]any
	disk := int64(4096)

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		mu.Lock()
		if patched == nil {
			patched = decoded
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"is_image_scan_enabled":   true,
			"scanner_state":           "ready",
			"scanner_disk_used_bytes": disk,
		})
	})
	mux.HandleFunc("GET /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"is_image_scan_enabled":   true,
			"scanner_state":           "ready",
			"scanner_disk_used_bytes": disk,
		})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.7.1"))
	defer srv.Close()

	const serverUUID = "bbbb0001-0001-4000-8000-000000000001"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_server_image_scanning", "test", `
				server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
				is_image_scan_enabled = true
			`),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "server_uuid", serverUUID),
				resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "is_image_scan_enabled", "true"),
				resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "scanner_state", "ready"),
				resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "scanner_disk_used_bytes", "4096"),
			),
		}},
	})

	mu.Lock()
	defer mu.Unlock()
	if patched == nil {
		t.Fatal("PATCH /image-scanning was not sent")
	}
	if patched["is_image_scan_enabled"] != true {
		t.Fatalf("is_image_scan_enabled = %v", patched["is_image_scan_enabled"])
	}
	for _, key := range []string{"image_scan_memory_mb", "image_scan_cpus", "image_scan_syft_timeout", "image_scan_grype_timeout"} {
		if _, ok := patched[key]; ok {
			t.Fatalf("4.7.1 create sent %s: %v", key, patched[key])
		}
	}
}

func TestServerImageScanningResource_Limits(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var patched map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		mu.Lock()
		if patched == nil {
			patched = decoded
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"is_image_scan_enabled":    true,
			"image_scan_memory_mb":     2048,
			"image_scan_syft_timeout":  1800,
			"image_scan_grype_timeout": 1200,
			"image_scan_cpus":          0.5,
			"scanner_state":            "ready",
		})
	})
	mux.HandleFunc("GET /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"is_image_scan_enabled":    true,
			"image_scan_memory_mb":     2048,
			"image_scan_syft_timeout":  1800,
			"image_scan_grype_timeout": 1200,
			"image_scan_cpus":          0.5,
			"scanner_state":            "ready",
		})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.7.5"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_server_image_scanning", "test", `
				server_uuid              = "bbbb0001-0001-4000-8000-000000000001"
				is_image_scan_enabled    = true
				image_scan_memory_mb     = 2048
				image_scan_syft_timeout  = 1800
				image_scan_grype_timeout = 1200
				image_scan_cpus          = 0.5
			`),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "image_scan_memory_mb", "2048"),
				resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "image_scan_syft_timeout", "1800"),
				resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "image_scan_grype_timeout", "1200"),
				resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "image_scan_cpus", "0.5"),
			),
		}},
	})

	mu.Lock()
	defer mu.Unlock()
	if patched["image_scan_memory_mb"] != float64(2048) || patched["image_scan_cpus"] != 0.5 {
		t.Fatalf("PATCH body = %v", patched)
	}
	if patched["image_scan_syft_timeout"] != float64(1800) || patched["image_scan_grype_timeout"] != float64(1200) {
		t.Fatalf("PATCH timeouts = %v", patched)
	}
}

func TestServerImageScanningResource_MemoryOn472(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var patched map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		mu.Lock()
		if patched == nil {
			patched = decoded
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"is_image_scan_enabled": true,
			"image_scan_memory_mb":  2048,
		})
	})
	mux.HandleFunc("GET /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"is_image_scan_enabled": true,
			"image_scan_memory_mb":  2048,
			"scanner_state":         "ready",
		})
	})

	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.7.2"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_server_image_scanning", "test", `
				server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
				is_image_scan_enabled = true
				image_scan_memory_mb  = 2048
			`),
			Check: resource.TestCheckResourceAttr("coolify_server_image_scanning.test", "image_scan_memory_mb", "2048"),
		}},
	})

	mu.Lock()
	defer mu.Unlock()
	if patched["image_scan_memory_mb"] != float64(2048) {
		t.Fatalf("PATCH body = %v", patched)
	}
	for _, key := range []string{"image_scan_cpus", "image_scan_syft_timeout", "image_scan_grype_timeout"} {
		if _, ok := patched[key]; ok {
			t.Fatalf("4.7.2 memory update sent %s: %v", key, patched[key])
		}
	}
}

func TestServerImageScanningResource_MemoryNeeds472(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		t.Error("PATCH sent to Coolify 4.7.1")
		http.Error(w, `{"message":"no"}`, http.StatusBadRequest)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_server_image_scanning", "test", `
				server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
				is_image_scan_enabled = true
				image_scan_memory_mb  = 2048
			`),
			ExpectError: regexp.MustCompile(`Coolify v4\.7\.2`),
		}},
	})
}

func TestServerImageScanningResource_RuntimeLimitsNeed473(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		t.Error("PATCH sent to Coolify 4.7.2")
		http.Error(w, `{"message":"no"}`, http.StatusBadRequest)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.7.2"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_server_image_scanning", "test", `
				server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
				is_image_scan_enabled = true
				image_scan_cpus       = 1
			`),
			ExpectError: regexp.MustCompile(`Coolify v4\.7\.3`),
		}},
	})
}

func TestServerImageScanningResource_RejectsUnknownLimit(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.7.5"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_server_image_scanning", "test", `
				server_uuid             = "bbbb0001-0001-4000-8000-000000000001"
				image_scan_syft_timeout = 45
			`),
			ExpectError: regexp.MustCompile(`image_scan_syft_timeout`),
		}},
	})
}

func TestServerImageScanningResource_Disappears(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	gone := false
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"is_image_scan_enabled":true}`))
	})
	mux.HandleFunc("GET /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		missing := gone
		mu.Unlock()
		if missing {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"is_image_scan_enabled":true,"scanner_state":"ready"}`))
	})
	mux.HandleFunc("DELETE /api/v1/servers/{uuid}/image-scanning", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gone = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.7.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.TestResourceConfig(srv.URL, "coolify_server_image_scanning", "test", `
				server_uuid           = "bbbb0001-0001-4000-8000-000000000001"
				is_image_scan_enabled = true
			`),
			Check: resource.ComposeAggregateTestCheckFunc(
				acctest.CheckPathDisappears(srv.URL, "/api/v1/servers/bbbb0001-0001-4000-8000-000000000001/image-scanning"),
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

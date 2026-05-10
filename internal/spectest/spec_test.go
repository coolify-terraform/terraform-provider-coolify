package spectest

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadSpec(t *testing.T) {
	t.Parallel()
	doc, err := LoadSpec("coolify-v4")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	model, err := (*doc).BuildV3Model()
	if err != nil {
		t.Fatalf("BuildV3Model error: %v", err)
	}
	if model == nil {
		t.Fatal("model is nil")
	}
	paths := model.Model.Paths
	if paths == nil || paths.PathItems.Len() == 0 {
		t.Fatal("no paths in spec")
	}

	// Verify a known endpoint exists.
	versionPath := paths.PathItems.GetOrZero("/version")
	if versionPath == nil {
		t.Fatal("expected /version path in spec")
	}
	if versionPath.Get == nil {
		t.Fatal("expected GET operation on /version")
	}
}

func TestValidatingHandler_ProjectCreate(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"uuid":"test-uuid-123"}`))
	})
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`v4.0.0`))
	})

	srv := httptest.NewServer(WithSpecValidation(t, "coolify-v4", mux))
	defer srv.Close()

	// Send a valid project create request.
	body := bytes.NewBufferString(`{"name":"test-project","description":"desc"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/projects", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestSpecVersions(t *testing.T) {
	t.Parallel()
	versions, err := SpecVersions()
	if err != nil {
		t.Fatalf("SpecVersions: %v", err)
	}
	if len(versions) == 0 {
		t.Fatal("expected at least one spec version")
	}
	found := false
	for _, v := range versions {
		if v == "coolify-v4" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected coolify-v4 in versions, got %v", versions)
	}
}

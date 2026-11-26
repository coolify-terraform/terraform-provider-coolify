//go:build !ci_app_a

package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Create
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Create(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:                    "docker-app-uuid",
		Name:                    "nginx-proxy",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/dockerimage missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		// Default true matches Coolify $request->boolean('autogenerate_domain', true).
		if v, ok := body["autogenerate_domain"].(bool); !ok || !v {
			t.Errorf("POST create expected autogenerate_domain=true by default, got %v", body["autogenerate_domain"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "uuid", "docker-app-uuid"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "name", "nginx-proxy"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "nginx:latest"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "ports_exposes", "80"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "environment_name", "production"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "autogenerate_domain", "true"),
				),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Update
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Update(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	currentApp := client.Application{
		UUID:                    "docker-upd-uuid",
		Name:                    "nginx-proxy",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/dockerimage missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentApp.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	var patchCount atomic.Int32
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		requestBody, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		n := patchCount.Add(1)
		wantName, wantTag := "nginx", "1.25"
		if n >= 2 {
			wantTag = "latest"
		}
		name, ok := requestBody["docker_registry_image_name"].(string)
		switch {
		case !ok:
			t.Errorf("PATCH missing docker_registry_image_name")
		case dockerImageUpdateNameHasTag(name):
			// Coolify update uses dockerImageNameRules(); a tagged name is 422.
			http.Error(w, `{"message":"The docker_registry_image_name field format is invalid."}`, http.StatusUnprocessableEntity)
			return
		case name != wantName:
			t.Errorf("PATCH docker_registry_image_name = %q, want %q", name, wantName)
		default:
			currentApp.DockerRegistryImageName = name
		}
		tag, ok := requestBody["docker_registry_image_tag"].(string)
		switch {
		case !ok:
			t.Errorf("PATCH missing docker_registry_image_tag")
		case tag != wantTag:
			t.Errorf("PATCH docker_registry_image_tag = %q, want %q", tag, wantTag)
		default:
			currentApp.DockerRegistryImageTag = tag
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "nginx:latest"),
				),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:1.25"
					ports_exposes  = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "nginx:1.25"),
				),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx"
					ports_exposes  = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "nginx"),
				),
			},
		},
	})
}

func TestDockerImageApplicationResource_UpdateKeepsExplicitTag(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	currentApp := client.Application{
		UUID:                    "docker-explicit-tag-uuid",
		Name:                    "nginx-proxy",
		DockerRegistryImageName: "nginx",
		DockerRegistryImageTag:  "latest",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := decodeRequestBodyMap(t, w, r); !ok {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentApp.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	var patchCount atomic.Int32
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		requestBody, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		n := patchCount.Add(1)
		wantName := "nginx"
		if n >= 2 {
			wantName = "redis"
		}
		name, ok := requestBody["docker_registry_image_name"].(string)
		switch {
		case !ok:
			t.Errorf("PATCH missing docker_registry_image_name")
		case dockerImageUpdateNameHasTag(name):
			http.Error(w, `{"message":"The docker_registry_image_name field format is invalid."}`, http.StatusUnprocessableEntity)
			return
		case name != wantName:
			t.Errorf("PATCH docker_registry_image_name = %q, want %q", name, wantName)
		default:
			currentApp.DockerRegistryImageName = name
		}
		tag, hasTag := requestBody["docker_registry_image_tag"].(string)
		switch {
		case tag == "latest":
			t.Errorf("PATCH overwrote explicit docker_registry_image_tag with latest")
		case n == 1 && !hasTag:
			t.Errorf("PATCH missing docker_registry_image_tag on first update")
		case hasTag && tag != "1.27-alpine":
			t.Errorf("PATCH docker_registry_image_tag = %q, want 1.27-alpine", tag)
		case hasTag:
			currentApp.DockerRegistryImageTag = tag
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name                     = "nginx-proxy"
					project_uuid             = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid              = "bbbb0002-0002-4000-8000-000000000002"
					docker_image             = "nginx"
					docker_registry_image_tag = "1.27-alpine"
					ports_exposes            = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "nginx"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_registry_image_tag", "1.27-alpine"),
				),
			},
			{
				// Name-only change; tag is already configured and must stay.
				Config: testDockerImageResourceConfig(srv.URL, `
					name                      = "nginx-proxy"
					project_uuid              = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid               = "bbbb0002-0002-4000-8000-000000000002"
					docker_image              = "redis"
					docker_registry_image_tag = "1.27-alpine"
					ports_exposes             = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "redis"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_registry_image_tag", "1.27-alpine"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Import
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Import(t *testing.T) {
	t.Parallel()
	// Simulate Coolify's tag stripping: the API may return "nginx" even though
	// the user configured "nginx:latest". The provider cannot reconstruct the
	// original tag, so docker_image is excluded from ImportStateVerify.
	app := client.Application{
		UUID:                    "aaaa0001-0001-4000-8000-000000000001",
		Name:                    "imported-docker-app",
		DockerRegistryImageName: "nginx", // Coolify strips the tag
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/dockerimage missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "imported-docker-app"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
			},
			{
				ResourceName:                         "coolify_application_docker_image.test",
				ImportState:                          true,
				ImportStateId:                        "aaaa0001-0001-4000-8000-000000000001",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"environment_name", "docker_image"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Disappears
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Disappears(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	appUUID := "docker-disappear-uuid"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/dockerimage missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": appUUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Application{
			UUID:                    appUUID,
			Name:                    "disappearing-docker",
			DockerRegistryImageName: "nginx:latest",
			PortsExposes:            "80",
			ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
			ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
			EnvironmentName:         "production",
		})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					project_uuid  = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid   = "bbbb0002-0002-4000-8000-000000000002"
					docker_image  = "nginx:latest"
					ports_exposes = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("coolify_application_docker_image.test", "uuid"),
					acctest.CheckResourceDisappears(srv.URL, "coolify_application_docker_image.test", "/api/v1/applications/"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Status
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Status(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:                    "docker-status-uuid",
		Name:                    "status-app",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
		Status:                  "running:healthy",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/dockerimage missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "status-app"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "status", "running:healthy"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_Timeouts
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_Timeouts(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:                    "docker-timeout-uuid",
		Name:                    "timeout-app",
		DockerRegistryImageName: "nginx:latest",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/dockerimage missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "timeout-app"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:latest"
					ports_exposes  = "80"
					timeouts = {
						create = "30m"
					}
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "uuid", "docker-timeout-uuid"),
				),
			},
		},
	})
}

func TestDockerImageApplicationResource_RejectsGiteaAppUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: testDockerImageResourceConfig(srv.URL, `
				project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
				server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
				docker_image   = "nginx:latest"
				ports_exposes  = "80"
				gitea_app_uuid = "cccc0001-0001-4000-8000-000000000001"
			`),
			ExpectError: regexp.MustCompile(`gitea_app_uuid is not valid for this application`),
		}},
	})
}

func TestDockerImageApplicationResource_InvalidPortsExposes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					project_uuid = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid  = "bbbb0002-0002-4000-8000-000000000002"
					docker_image = "nginx:latest"
					ports_exposes = "abc"
				`),
				ExpectError: regexp.MustCompile(`comma-separated list of\s+port\s+numbers`),
			},
		},
	})
}

func TestDockerImageApplicationResource_InvalidDomains(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.NotFoundHandler()))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					project_uuid = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid  = "bbbb0002-0002-4000-8000-000000000002"
					docker_image = "nginx:latest"
					ports_exposes = "3000"
					domains = "app.example.com"
				`),
				ExpectError: regexp.MustCompile(`must be empty, or comma-separated http:// or https:// URLs`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_CreateReadBackFailurePreservesState
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_CreateReadBackFailurePreservesState(t *testing.T) {
	t.Parallel()

	const appUUID = "docker-readback-fail-uuid"

	var forceReadFailure atomic.Bool
	var posts atomic.Int32
	var deletes atomic.Int32
	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/dockerimage missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		if body["docker_registry_image_name"] != "nginx:latest" || body["ports_exposes"] != "80" {
			http.Error(w, `{"error":"unexpected application"}`, http.StatusUnprocessableEntity)
			return
		}
		// Arm only the first create. Retries must keep failing, and the
		// tainted replace must be readable. Clear deleted so that replace
		// is not a 404 left over from destroy. Delete polls until GET is
		// not found.
		if posts.Add(1) == 1 {
			forceReadFailure.Store(true)
		}
		mu.Lock()
		deleted = false
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": appUUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		if forceReadFailure.Load() {
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Application{
			UUID:                    appUUID,
			Name:                    "docker-readback",
			DockerRegistryImageName: "nginx:latest",
			PortsExposes:            "80",
			ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
			ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
			EnvironmentName:         "production",
		})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != appUUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		deletes.Add(1)
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	config := testDockerImageResourceConfig(srv.URL, `
		project_uuid  = "aaaa0002-0002-4000-8000-000000000002"
		server_uuid   = "bbbb0002-0002-4000-8000-000000000002"
		docker_image  = "nginx:latest"
		ports_exposes = "80"
	`)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`(?s)Application created but refresh failed.*Could not read application.*partial Terraform state was saved`),
			},
			{
				PreConfig: func() { forceReadFailure.Store(false) },
				Config:    config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "nginx:latest"),
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "ports_exposes", "80"),
					func(*terraform.State) error {
						if posts.Load() != 2 {
							return fmt.Errorf("posts = %d, want 2", posts.Load())
						}
						if deletes.Load() != 1 {
							return fmt.Errorf("deletes = %d, want 1 (state was not saved if this is 0)", deletes.Load())
						}
						return nil
					},
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// TestDockerImageApplicationResource_LatestTagNormalization verifies that when
// Coolify strips the ":latest" tag from a Docker image name (e.g. returns
// "alpine" instead of "alpine:latest"), the provider preserves the user's
// original value and does not show a perpetual diff.
func TestDockerImageApplicationResource_LatestTagNormalization(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:                    "docker-latest-uuid",
		Name:                    "alpine-test",
		DockerRegistryImageName: "alpine", // API strips :latest
		PortsExposes:            "8080",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/dockerimage missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// User provides "alpine:latest", API returns "alpine"
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "alpine-test"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "alpine:latest"
					ports_exposes  = "8080"
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "alpine:latest"),
				),
			},
			{
				// Re-apply same config: should produce an empty plan (no diff)
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "alpine-test"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "alpine:latest"
					ports_exposes  = "8080"
				`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestDockerImageApplicationResource_RedeployOnUpdate(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	currentApp := client.Application{
		UUID:                    "docker-redeploy-uuid",
		Name:                    "web",
		DockerRegistryImageName: "nginx:1.24",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}
	var restartCalled atomic.Bool
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		for _, field := range []string{"project_uuid", "server_uuid"} {
			if _, exists := body[field]; !exists {
				t.Errorf("POST /api/v1/applications/dockerimage missing required field %q", field)
				http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"uuid": currentApp.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requestBody, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if v, ok := requestBody["docker_registry_image_name"].(string); ok {
			currentApp.DockerRegistryImageName = v
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("POST /api/v1/applications/{uuid}/restart", func(w http.ResponseWriter, _ *http.Request) {
		restartCalled.Store(true)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Restarting."})
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name              = "web"
					project_uuid      = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid       = "bbbb0002-0002-4000-8000-000000000002"
					docker_image      = "nginx:1.24"
					ports_exposes     = "80"
					redeploy_on_update = true
				`),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name              = "web"
					project_uuid      = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid       = "bbbb0002-0002-4000-8000-000000000002"
					docker_image      = "nginx:1.25"
					ports_exposes     = "80"
					redeploy_on_update = true
				`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_application_docker_image.test", "docker_image", "nginx:1.25"),
				),
			},
		},
	})
	if !restartCalled.Load() {
		t.Error("expected restart to be called when docker_image changed with redeploy_on_update=true")
	}
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_CreateAPIError
// ---------------------------------------------------------------------------

func TestDockerImageApplicationResource_CreateAPIError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"validation failed"}`, http.StatusUnprocessableEntity)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					project_uuid  = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid   = "bbbb0002-0002-4000-8000-000000000002"
					docker_image  = "nginx"
					ports_exposes = "80"
				`),
				ExpectError: regexp.MustCompile(`Error creating docker image application`),
			},
		},
	})
}

func testDockerImageResourceConfig(endpoint, attrs string) string {
	return acctest.TestResourceConfig(endpoint, "coolify_application_docker_image", "test", attrs)
}

// dockerImageUpdateNameHasTag mirrors Coolify dockerImageNameRules: a tag
// after the last colon is invalid on PATCH docker_registry_image_name.
// registry:port/name is not a tag.
func dockerImageUpdateNameHasTag(name string) bool {
	i := strings.LastIndex(name, ":")
	return i >= 0 && !strings.Contains(name[i+1:], "/")
}

// TestDockerImageApplicationResource_AutogenerateDomainFalse asserts create body
// sends autogenerate_domain=false and state keeps domains empty (no generation).
func TestDockerImageApplicationResource_AutogenerateDomainFalse(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:                    "docker-nogen-uuid",
		Name:                    "internal-worker",
		DockerRegistryImageName: "nginx:alpine",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
		Domains:                 "",
	}
	mu := sync.Mutex{}
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		v, ok := body["autogenerate_domain"].(bool)
		if !ok || v {
			t.Errorf("expected autogenerate_domain=false, got %v", body["autogenerate_domain"])
		}
		if d, exists := body["domains"]; exists && d != nil && d != "" {
			t.Errorf("expected domains omitted or empty on create, got %v", d)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{{
			Config: testDockerImageResourceConfig(srv.URL, `
				name                = "internal-worker"
				project_uuid        = "aaaa0002-0002-4000-8000-000000000002"
				server_uuid         = "bbbb0002-0002-4000-8000-000000000002"
				docker_image        = "nginx:alpine"
				ports_exposes       = "80"
				autogenerate_domain = false
			`),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("coolify_application_docker_image.test", "autogenerate_domain", "false"),
				resource.TestCheckResourceAttr("coolify_application_docker_image.test", "domains", ""),
			),
		}},
	})
}

// TestDockerImageApplicationResource_ClearDomainsOnUpdate asserts empty domains
// is sent on PATCH so Coolify can clear fqdn (#645).
func TestDockerImageApplicationResource_ClearDomainsOnUpdate(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	currentApp := client.Application{
		UUID:                    "docker-clear-dom-uuid",
		Name:                    "nginx-proxy",
		DockerRegistryImageName: "nginx:alpine",
		PortsExposes:            "80",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
		Domains:                 "https://app.example.com",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": currentApp.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != currentApp.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		// Clearing FQDN requires domains present as "" (not omitted via omitempty).
		d, exists := body["domains"]
		if !exists {
			t.Error("PATCH expected domains key when clearing FQDN")
		} else if s, ok := d.(string); !ok || s != "" {
			t.Errorf("PATCH expected domains=\"\", got %v (%T)", d, d)
		} else {
			currentApp.Domains = s
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(currentApp)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:alpine"
					ports_exposes  = "80"
					domains        = "https://app.example.com"
				`),
				Check: resource.TestCheckResourceAttr("coolify_application_docker_image.test", "domains", "https://app.example.com"),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "nginx-proxy"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "nginx:alpine"
					ports_exposes  = "80"
					domains        = ""
				`),
				Check: resource.TestCheckResourceAttr("coolify_application_docker_image.test", "domains", ""),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_CreateWithoutPortsExposes
// ---------------------------------------------------------------------------

// TestDockerImageApplicationResource_CreateWithoutPortsExposes asserts that a
// zero-port app (ports_exposes omitted) is created without the ports_exposes
// key in the POST body, that the state stays null, and that the plan is clean.
func TestDockerImageApplicationResource_CreateWithoutPortsExposes(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	app := client.Application{
		UUID:                    "docker-zeroport-uuid",
		Name:                    "gitlab-runner",
		DockerRegistryImageName: "gitlab/gitlab-runner",
		PortsExposes:            "",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if _, exists := body["ports_exposes"]; exists {
			t.Errorf("POST /api/v1/applications/dockerimage must omit ports_exposes for a zero-port app, got %v", body["ports_exposes"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name         = "gitlab-runner"
					project_uuid = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid  = "bbbb0002-0002-4000-8000-000000000002"
					docker_image = "gitlab/gitlab-runner"
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_application_docker_image.test", tfjsonpath.New("ports_exposes"), knownvalue.Null()),
				},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_ClearPortsExposesOnUpdate
// ---------------------------------------------------------------------------

// TestDockerImageApplicationResource_ClearPortsExposesOnUpdate asserts the
// PATCH wire format for ports_exposes: omitted when unchanged, an explicit
// JSON null when cleared via "" or omission, and the state converging to the
// configured form ("" stays "", omission becomes null).
func TestDockerImageApplicationResource_ClearPortsExposesOnUpdate(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	app := client.Application{
		UUID:                    "docker-clrport-uuid",
		Name:                    "gitlab-runner",
		DockerRegistryImageName: "gitlab/gitlab-runner",
		PortsExposes:            "9252",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if p, exists := body["ports_exposes"]; !exists || p != "9252" {
			t.Errorf("POST ports_exposes = %v (exists=%v), want \"9252\"", p, exists)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	var patchCount atomic.Int32
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		n := patchCount.Add(1)
		switch n {
		case 1:
			// Name-only change: ports_exposes must be omitted (no change).
			if _, exists := body["ports_exposes"]; exists {
				t.Errorf("PATCH #1 (name change) must omit ports_exposes, got %v", body["ports_exposes"])
			}
			if name, ok := body["name"].(string); ok {
				app.Name = name
			}
		case 2:
			// ports_exposes = "": Coolify rejects "", so the wire form is null.
			v, exists := body["ports_exposes"]
			if !exists || v != nil {
				t.Errorf("PATCH #2 (ports_exposes = \"\") must send explicit null, got %v (exists=%v)", v, exists)
			}
			app.PortsExposes = ""
		case 3:
			// Omission: zero-to-zero ("" -> null) is a no-op, key omitted.
			if _, exists := body["ports_exposes"]; exists {
				t.Errorf("PATCH #3 (ports_exposes omitted) must omit ports_exposes, got %v", body["ports_exposes"])
			}
			app.PortsExposes = ""
		default:
			t.Errorf("unexpected PATCH #%d", n)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	// The clear steps (PATCH null) need v4.3.15; run the whole test at that
	// boundary so the gate is exercised at its minimum.
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "v4.3.15"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "gitlab-runner"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "gitlab/gitlab-runner"
					ports_exposes  = "9252"
				`),
				Check: resource.TestCheckResourceAttr("coolify_application_docker_image.test", "ports_exposes", "9252"),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "gitlab-runner-2"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "gitlab/gitlab-runner"
					ports_exposes  = "9252"
				`),
				Check: resource.TestCheckResourceAttr("coolify_application_docker_image.test", "name", "gitlab-runner-2"),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "gitlab-runner-2"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "gitlab/gitlab-runner"
					ports_exposes  = ""
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_application_docker_image.test", tfjsonpath.New("ports_exposes"), knownvalue.StringExact("")),
				},
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name         = "gitlab-runner-2"
					project_uuid = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid  = "bbbb0002-0002-4000-8000-000000000002"
					docker_image = "gitlab/gitlab-runner"
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_application_docker_image.test", tfjsonpath.New("ports_exposes"), knownvalue.Null()),
				},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_CreateWithEmptyPortsExposes
// ---------------------------------------------------------------------------

// TestDockerImageApplicationResource_CreateWithEmptyPortsExposes asserts that
// ports_exposes = "" takes the same omitempty path as omitting the attribute:
// the POST body carries no ports_exposes key, and the state keeps the
// configured "".
func TestDockerImageApplicationResource_CreateWithEmptyPortsExposes(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	app := client.Application{
		UUID:                    "docker-emptyport-uuid",
		Name:                    "gitlab-runner",
		DockerRegistryImageName: "gitlab/gitlab-runner",
		PortsExposes:            "",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if _, exists := body["ports_exposes"]; exists {
			t.Errorf("POST /api/v1/applications/dockerimage must omit ports_exposes when it is \"\", got %v", body["ports_exposes"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "gitlab-runner"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "gitlab/gitlab-runner"
					ports_exposes  = ""
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_application_docker_image.test", tfjsonpath.New("ports_exposes"), knownvalue.StringExact("")),
				},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_ImportEmptyPortsExposes
// ---------------------------------------------------------------------------

// TestDockerImageApplicationResource_ImportEmptyPortsExposes asserts that
// importing an application whose API ports_exposes is empty stores null in
// state: the lenient flatten leaves an empty API value alone.
func TestDockerImageApplicationResource_ImportEmptyPortsExposes(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	// The mock always reports an empty ports_exposes, simulating a zero-port
	// app stored in Coolify.
	app := client.Application{
		UUID:                    "cccc0002-0002-4000-8000-000000000002",
		Name:                    "gitlab-runner",
		DockerRegistryImageName: "gitlab/gitlab-runner",
		PortsExposes:            "",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if body["ports_exposes"] != "9252" {
			t.Errorf("POST /api/v1/applications/dockerimage ports_exposes = %v, want \"9252\"", body["ports_exposes"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				// Create with a port. The mock stores an empty ports_exposes,
				// so the read-back preserves the configured "9252" in state.
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "gitlab-runner"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "gitlab/gitlab-runner"
					ports_exposes  = "9252"
				`),
			},
			{
				// Re-import the same app. The API reports an empty
				// ports_exposes, so a fresh import must store null. A null
				// string attribute is absent from the flatmap.
				ResourceName:  "coolify_application_docker_image.test",
				ImportState:   true,
				ImportStateId: app.UUID,
				Check: resource.TestCheckFunc(func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["coolify_application_docker_image.test"]
					if !ok {
						return fmt.Errorf("resource coolify_application_docker_image.test not found in imported state")
					}
					if v, exists := rs.Primary.Attributes["ports_exposes"]; exists {
						return fmt.Errorf("imported ports_exposes = %q, want null (absent)", v)
					}
					return nil
				}),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_ClearPortsExposesRejectedOnOldCoolify
// ---------------------------------------------------------------------------

// TestDockerImageApplicationResource_ClearPortsExposesRejectedOnOldCoolify
// asserts the clear gate: on a Coolify older than v4.3.15 (the unit-test
// default is v4.2.0), clearing a stored ports_exposes is rejected at plan
// time instead of sending a PATCH null that the API would 422.
func TestDockerImageApplicationResource_ClearPortsExposesRejectedOnOldCoolify(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	app := client.Application{
		UUID:                    "docker-rejectport-uuid",
		Name:                    "gitlab-runner",
		DockerRegistryImageName: "gitlab/gitlab-runner",
		PortsExposes:            "9252",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if body["ports_exposes"] != "9252" {
			t.Errorf("POST /api/v1/applications/dockerimage ports_exposes = %v, want \"9252\"", body["ports_exposes"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	// Default version endpoint reports v4.2.0-test, below the v4.3.15 clear gate.
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "gitlab-runner"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "gitlab/gitlab-runner"
					ports_exposes  = "9252"
				`),
				Check: resource.TestCheckResourceAttr("coolify_application_docker_image.test", "ports_exposes", "9252"),
			},
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "gitlab-runner"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "gitlab/gitlab-runner"
					ports_exposes  = ""
				`),
				ExpectError: regexp.MustCompile(`older than v4\.3\.15`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_CreateZeroPortsRejectedOnOldCoolify
// ---------------------------------------------------------------------------

// TestDockerImageApplicationResource_CreateZeroPortsRejectedOnOldCoolify
// asserts the create gate: on a Coolify older than v4.1.2, creating a
// zero-port docker image app is rejected at plan time instead of sending
// a POST with an absent ports_exposes that the API would 422.
func TestDockerImageApplicationResource_CreateZeroPortsRejectedOnOldCoolify(t *testing.T) {
	t.Parallel()

	// v4.1.1: below the v4.1.2 create gate. No app endpoints are needed:
	// the plan fails before any application request.
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.NewServeMux(), "v4.1.1"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testDockerImageResourceConfig(srv.URL, `
					name         = "gitlab-runner"
					project_uuid = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid  = "bbbb0002-0002-4000-8000-000000000002"
					docker_image = "gitlab/gitlab-runner"
				`),
				ExpectError: regexp.MustCompile(`older than v4\.1\.2`),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TestDockerImageApplicationResource_ZeroToZeroUpdateOmitsKey
// ---------------------------------------------------------------------------

// TestDockerImageApplicationResource_ZeroToZeroUpdateOmitsKey asserts that
// transitions between the two zero-port encodings ("" and null) send no
// ports_exposes key at all: a stored port only exists when the state is
// non-empty, so these transitions are no-ops, and the explicit null they
// would otherwise send 422s below v4.3.15 (the unit-test default is
// v4.2.0-test, where the gate must not fire either).
func TestDockerImageApplicationResource_ZeroToZeroUpdateOmitsKey(t *testing.T) {
	t.Parallel()
	mu := sync.Mutex{}
	deleted := false
	app := client.Application{
		UUID:                    "docker-zptrans-uuid",
		Name:                    "gitlab-runner",
		DockerRegistryImageName: "gitlab/gitlab-runner",
		PortsExposes:            "",
		ProjectUUID:             "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:              "bbbb0002-0002-4000-8000-000000000002",
		EnvironmentName:         "production",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/applications/dockerimage", func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if _, exists := body["ports_exposes"]; exists {
			t.Errorf("POST /api/v1/applications/dockerimage must omit ports_exposes for a zero-port app, got %v", body["ports_exposes"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": app.UUID})
	})
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("PATCH /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("uuid") != app.UUID {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		body, ok := decodeRequestBodyMap(t, w, r)
		if !ok {
			return
		}
		if _, exists := body["ports_exposes"]; exists {
			t.Errorf("PATCH must omit ports_exposes for a zero-to-zero transition, got %v", body["ports_exposes"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	})
	mux.HandleFunc("DELETE /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		CheckDestroy:             acctest.CheckDestroy(srv.URL, "coolify_application_docker_image", "/api/v1/applications/"),
		Steps: []resource.TestStep{
			{
				// Create with "" (state keeps the configured "").
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "gitlab-runner"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "gitlab/gitlab-runner"
					ports_exposes  = ""
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_application_docker_image.test", tfjsonpath.New("ports_exposes"), knownvalue.StringExact("")),
				},
			},
			{
				// "" -> null: no-op, must omit the key.
				Config: testDockerImageResourceConfig(srv.URL, `
					name         = "gitlab-runner"
					project_uuid = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid  = "bbbb0002-0002-4000-8000-000000000002"
					docker_image = "gitlab/gitlab-runner"
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_application_docker_image.test", tfjsonpath.New("ports_exposes"), knownvalue.Null()),
				},
			},
			{
				// null -> "": no-op, must omit the key.
				Config: testDockerImageResourceConfig(srv.URL, `
					name           = "gitlab-runner"
					project_uuid   = "aaaa0002-0002-4000-8000-000000000002"
					server_uuid    = "bbbb0002-0002-4000-8000-000000000002"
					docker_image   = "gitlab/gitlab-runner"
					ports_exposes  = ""
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_application_docker_image.test", tfjsonpath.New("ports_exposes"), knownvalue.StringExact("")),
				},
			},
		},
	})
}

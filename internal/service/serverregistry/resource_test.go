package serverregistry_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestServerDockerRegistryResource_LoginAndList(t *testing.T) {
	t.Parallel()
	const serverUUID = "bbbb0001-0001-4000-8000-000000000001"
	var mu sync.Mutex
	logins := map[string]string{}
	var lastLogin map[string]string
	var loggedOut string

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/"+serverUUID+"/registries", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode login: %v", err)
			http.Error(w, `{"message":"bad"}`, http.StatusBadRequest)
			return
		}
		mu.Lock()
		lastLogin = body
		logins[body["registry"]] = body["username"]
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Logged in."})
	})
	mux.HandleFunc("GET /api/v1/servers/"+serverUUID+"/registries", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		rows := make([]map[string]interface{}, 0, len(logins))
		for reg, user := range logins {
			rows = append(rows, map[string]interface{}{
				"registry": reg, "logged_in": true, "username": user, "source": "docker-config",
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"registries": rows})
	})
	mux.HandleFunc("DELETE /api/v1/servers/"+serverUUID+"/registries/{registry}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		loggedOut = r.PathValue("registry")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"Logged out."}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.4.0"))
	defer srv.Close()

	config := func(user, pass, registry string) string {
		return acctest.ProviderBlockForURL(srv.URL) + fmt.Sprintf(`
resource "coolify_server_docker_registry" "test" {
  server_uuid = %q
  registry    = %q
  username    = %q
  password    = %q
}
data "coolify_server_docker_registries" "all" {
  server_uuid = %q
  depends_on  = [coolify_server_docker_registry.test]
}
`, serverUUID, registry, user, pass, serverUUID)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("octocat", "change-me-in-production", "https://ghcr.io/v2/"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_server_docker_registry.test", "registry", "https://ghcr.io/v2/"),
					resource.TestCheckResourceAttr("coolify_server_docker_registry.test", "username", "octocat"),
					resource.TestCheckResourceAttr("data.coolify_server_docker_registries.all", "registries.0.logged_in", "true"),
					func(*terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if lastLogin["registry"] != "ghcr.io" || lastLogin["password"] != "change-me-in-production" {
							return fmt.Errorf("login body = %v", lastLogin)
						}
						return nil
					},
				),
			},
			{Config: config("octocat", "change-me-in-production", "https://ghcr.io/v2/"), PlanOnly: true, ExpectNonEmptyPlan: false},
			{
				Config: config("octocat2", "change-me-in-production", "ghcr.io"),
				Check:  resource.TestCheckResourceAttr("coolify_server_docker_registry.test", "username", "octocat2"),
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if loggedOut == "" {
		t.Fatal("destroy did not log out")
	}
}

func TestServerDockerRegistryResource_DeleteNotFound(t *testing.T) {
	t.Parallel()
	const serverUUID = "bbbb0001-0001-4000-8000-000000000002"
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/"+serverUUID+"/registries", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"Logged in."}`))
	})
	mux.HandleFunc("GET /api/v1/servers/"+serverUUID+"/registries", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"registries":[{"registry":"ghcr.io","logged_in":true,"username":"octocat","source":"docker-config"}]}`))
	})
	mux.HandleFunc("DELETE /api/v1/servers/"+serverUUID+"/registries/{registry}", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.4.0"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: registryTestConfig(srv.URL, serverUUID, "ghcr.io")},
			acctest.DestroyRemoveResourceStep(srv.URL),
		},
	})
}

func TestServerDockerRegistryResource_ReadErrorKeepsState(t *testing.T) {
	t.Parallel()
	const serverUUID = "bbbb0001-0001-4000-8000-000000000003"
	var mu sync.Mutex
	failRead := false
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/"+serverUUID+"/registries", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"Logged in."}`))
	})
	mux.HandleFunc("GET /api/v1/servers/"+serverUUID+"/registries", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		if failRead {
			// HTTP 200 with a non-empty error means the server could not be read.
			_, _ = w.Write([]byte(`{"registries":[],"error":"could not read docker config"}`))
			return
		}
		_, _ = w.Write([]byte(`{"registries":[{"registry":"ghcr.io","logged_in":true,"username":"octocat","source":"docker-config"}]}`))
	})
	mux.HandleFunc("DELETE /api/v1/servers/"+serverUUID+"/registries/{registry}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.4.0"))
	defer srv.Close()
	config := registryTestConfig(srv.URL, serverUUID, "ghcr.io")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr("coolify_server_docker_registry.test", "username", "octocat"),
			},
			{
				PreConfig: func() {
					mu.Lock()
					failRead = true
					mu.Unlock()
				},
				Config:      config,
				ExpectError: regexp.MustCompile(`reading Docker registry login`),
			},
			{
				PreConfig: func() {
					mu.Lock()
					failRead = false
					mu.Unlock()
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestServerDockerRegistryResource_DisappearsWhenLoggedOut(t *testing.T) {
	t.Parallel()
	const serverUUID = "bbbb0001-0001-4000-8000-000000000004"
	var mu sync.Mutex
	loggedIn := true
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/servers/"+serverUUID+"/registries", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"Logged in."}`))
	})
	mux.HandleFunc("GET /api/v1/servers/"+serverUUID+"/registries", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		_, _ = fmt.Fprintf(w, `{"registries":[{"registry":"ghcr.io","logged_in":%t,"username":"octocat","source":"docker-config"}]}`, loggedIn)
	})
	mux.HandleFunc("DELETE /api/v1/servers/"+serverUUID+"/registries/{registry}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.4.0"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: registryTestConfig(srv.URL, serverUUID, "ghcr.io"),
				Check: func(*terraform.State) error {
					mu.Lock()
					loggedIn = false
					mu.Unlock()
					return nil
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestServerDockerRegistriesDataSource_ReadError(t *testing.T) {
	t.Parallel()
	const serverUUID = "bbbb0001-0001-4000-8000-000000000005"
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/servers/"+serverUUID+"/registries", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"registries":[],"error":"could not read docker config"}`))
	})
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(mux, "4.4.0"))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + fmt.Sprintf(`
data "coolify_server_docker_registries" "all" {
  server_uuid = %q
}
`, serverUUID),
				ExpectError: regexp.MustCompile(`Error listing Docker registries`),
			},
		},
	})
}

func registryTestConfig(serverURL, serverUUID, registry string) string {
	return acctest.ProviderBlockForURL(serverURL) + fmt.Sprintf(`
resource "coolify_server_docker_registry" "test" {
  server_uuid = %q
  registry    = %q
  username    = "octocat"
  password    = "change-me-in-production"
}
`, serverUUID, registry)
}

func TestServerDockerRegistryResource_RejectsOldCoolify(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(acctest.WithVersionEndpointVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}), "4.3.23"))
	defer srv.Close()
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_server_docker_registry" "test" {
  server_uuid = "bbbb0001-0001-4000-8000-000000000001"
  registry    = "ghcr.io"
  username    = "octocat"
  password    = "change-me-in-production"
}
`,
			ExpectError: regexp.MustCompile(`requires Coolify 4.4`),
		}},
	})
}

//go:build !ci_app_b

package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApplicationDataSource_Read(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:                             "cccc0004-0004-4000-8000-000000000001",
		Name:                             "data-source-app",
		Description:                      "a test app",
		GitRepository:                    "https://github.com/example/repo",
		GitBranch:                        "main",
		BuildPack:                        "nixpacks",
		PortsExposes:                     "8080",
		ProjectUUID:                      "aaaa0002-0002-4000-8000-000000000002",
		ServerUUID:                       "bbbb0002-0002-4000-8000-000000000002",
		DockerComposeRaw:                 "version: '3'\nservices:\n  web:\n    image: nginx",
		DockerRegistryImageName:          "registry.example.com/app:latest",
		MaxRestartCount:                  func() *int64 { v := int64(10); return &v }(),
		RestartLimitReached:              func() *bool { v := false; return &v }(),
		ContainerPresent:                 func() *bool { v := true; return &v }(),
		IsConsistentContainerNameEnabled: func() *bool { v := true; return &v }(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "coolify" {
  endpoint  = %q
  token = "test-token"
}

data "coolify_application" "test" {
  uuid = "cccc0004-0004-4000-8000-000000000001"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_application.test", "uuid", "cccc0004-0004-4000-8000-000000000001"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "name", "data-source-app"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "description", "a test app"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "git_repository", "https://github.com/example/repo"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "git_branch", "main"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "build_pack", "nixpacks"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "ports_exposes", "8080"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "project_uuid", "aaaa0002-0002-4000-8000-000000000002"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "server_uuid", "bbbb0002-0002-4000-8000-000000000002"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "docker_compose_raw", "version: '3'\nservices:\n  web:\n    image: nginx"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "docker_registry_image_name", "registry.example.com/app:latest"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "max_restart_count", "10"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "restart_limit_reached", "false"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "container_present", "true"),
					resource.TestCheckResourceAttr("data.coolify_application.test", "is_consistent_container_name_enabled", "true"),
					resource.TestCheckNoResourceAttr("data.coolify_application.test", "domain_port_overrides.%"),
				),
			},
		},
	})
}

func TestApplicationDataSource_DomainPortOverrides(t *testing.T) {
	t.Parallel()
	overrides := map[string]int64{"https://app.example.com": 3000}
	app := client.Application{
		UUID:                "cccc0004-0004-4000-8000-000000000004",
		Name:                "domain-port-ds-app",
		DomainPortOverrides: &overrides,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

data "coolify_application" "test" {
  uuid = "cccc0004-0004-4000-8000-000000000004"
}
`, srv.URL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.coolify_application.test", "domain_port_overrides.%", "1"),
				resource.TestCheckResourceAttr("data.coolify_application.test", "domain_port_overrides.https://app.example.com", "3000"),
			),
		}},
	})
}

func TestApplicationDataSource_NoindexDomains(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID:           "cccc0004-0004-4000-8000-000000000003",
		Name:           "noindex-ds-app",
		NoindexDomains: []string{"https://alpha.example.com", "https://zebra.example.com"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

data "coolify_application" "test" {
  uuid = "cccc0004-0004-4000-8000-000000000003"
}
`, srv.URL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.coolify_application.test", "noindex_domains.#", "2"),
				resource.TestCheckResourceAttr("data.coolify_application.test", "noindex_domains.0", "https://alpha.example.com"),
				resource.TestCheckResourceAttr("data.coolify_application.test", "noindex_domains.1", "https://zebra.example.com"),
			),
		}},
	})
}

func TestApplicationDataSource_ConsistentContainerNameDefault(t *testing.T) {
	t.Parallel()
	app := client.Application{
		UUID: "cccc0004-0004-4000-8000-000000000002",
		Name: "default-consistent-name",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/applications/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
	})

	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
provider "coolify" {
  endpoint = %q
  token    = "test-token"
}

data "coolify_application" "test" {
  uuid = "cccc0004-0004-4000-8000-000000000002"
}
`, srv.URL),
			Check: resource.TestCheckResourceAttr("data.coolify_application.test", "is_consistent_container_name_enabled", "false"),
		}},
	})
}

func TestApplicationDataSource_NotFound(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_application" "test" {
  uuid = "00000000-0000-4000-8000-000000000000"
}`,
				ExpectError: regexp.MustCompile(`Error reading application`),
			},
		},
	})
}

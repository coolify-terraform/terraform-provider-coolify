package environment_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestEnvironmentListDataSource(t *testing.T) {
	t.Parallel()
	server, _ := newMockEnvironmentServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "first" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "first-env"
  description  = "first"
}

resource "coolify_environment" "second" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "second-env"
  description  = "second"
}

data "coolify_environments" "all" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  depends_on   = [coolify_environment.first, coolify_environment.second]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environments.all", "environments.#", "2"),
					environmentListHasUUID("data.coolify_environments.all", "coolify_environment.first"),
					environmentListHasUUID("data.coolify_environments.all", "coolify_environment.second"),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "first" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "first-env"
  description  = "first"
}

resource "coolify_environment" "second" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "second-env"
  description  = "second"
}

data "coolify_environments" "filtered" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  depends_on   = [coolify_environment.first, coolify_environment.second]
  filter {
    name   = "name"
    values = ["first-env"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environments.filtered", "environments.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_environments.filtered", "environments.0.name", "first-env"),
					resource.TestCheckResourceAttrPair(
						"data.coolify_environments.filtered", "environments.0.uuid",
						"coolify_environment.first", "uuid",
					),
				),
			},
			{
				Config: acctest.ProviderBlockForURL(server.URL) + `
resource "coolify_environment" "first" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "first-env"
  description  = "first"
}

resource "coolify_environment" "second" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  name         = "second-env"
  description  = "second"
}

data "coolify_environments" "by_uuid" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
  depends_on   = [coolify_environment.first, coolify_environment.second]
  filter {
    name   = "uuid"
    values = [coolify_environment.first.uuid]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.coolify_environments.by_uuid", "environments.#", "1"),
					resource.TestCheckResourceAttr("data.coolify_environments.by_uuid", "environments.0.name", "first-env"),
					resource.TestCheckResourceAttrPair(
						"data.coolify_environments.by_uuid", "environments.0.uuid",
						"coolify_environment.first", "uuid",
					),
				),
			},
		},
	})
}

func environmentListHasUUID(listName, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not in state", resourceName)
		}
		want := resourceState.Primary.Attributes["uuid"]
		if want == "" {
			return fmt.Errorf("resource %s has no uuid", resourceName)
		}
		listState, ok := s.RootModule().Resources[listName]
		if !ok {
			return fmt.Errorf("data source %s not in state", listName)
		}
		count, err := strconv.Atoi(listState.Primary.Attributes["environments.#"])
		if err != nil {
			return fmt.Errorf("environments count: %w", err)
		}
		for i := 0; i < count; i++ {
			if listState.Primary.Attributes[fmt.Sprintf("environments.%d.uuid", i)] == want {
				return nil
			}
		}
		return fmt.Errorf("uuid %s from %s not found in %s", want, resourceName, listName)
	}
}

func TestEnvironmentListDataSource_MissingUUID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/projects/{projectUUID}/environments", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"name":"production"}]`))
	})
	server := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: acctest.ProviderBlockForURL(server.URL) + `
data "coolify_environments" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
}
`,
			ExpectError: regexp.MustCompile(`without a uuid`),
		}},
	})
}

func TestEnvironmentListDataSource_APIError(t *testing.T) {
	t.Parallel()
	mockSrv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
	})))
	defer mockSrv.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderBlockForURL(mockSrv.URL) + `
data "coolify_environments" "test" {
  project_uuid = "aaaa0001-0001-4000-8000-000000000001"
}
`,
				ExpectError: regexp.MustCompile(`Error listing environments`),
			},
		},
	})
}

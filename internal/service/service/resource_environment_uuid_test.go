package service_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestServiceResource_EnvironmentUUIDCreateAndUpdate(t *testing.T) {
	t.Parallel()
	const svcUUID = "dddd0001-0001-4000-8000-000000000001"
	const envUUID = "eeee0001-0001-4000-8000-000000000001"
	var mu sync.Mutex
	var posts []map[string]any
	var patches []map[string]any
	description := ""
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/services", func(w http.ResponseWriter, r *http.Request) {
		body, err := decodeBody(r)
		if err != nil {
			http.Error(w, `{"message":"bad body"}`, http.StatusBadRequest)
			return
		}
		mu.Lock()
		posts = append(posts, body)
		deleted = false
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": svcUUID})
	})
	mux.HandleFunc("GET /api/v1/services/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if deleted {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		payload := map[string]any{
			"uuid":             svcUUID,
			"name":             "uuid-svc",
			"type":             "plausible",
			"project_uuid":     "aaaa0001-0001-4000-8000-000000000001",
			"server_uuid":      "bbbb0001-0001-4000-8000-000000000001",
			"environment_name": "staging",
		}
		if description != "" {
			payload["description"] = description
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})
	mux.HandleFunc("PATCH /api/v1/services/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		body, err := decodeBody(r)
		if err != nil {
			http.Error(w, `{"message":"bad body"}`, http.StatusBadRequest)
			return
		}
		mu.Lock()
		patches = append(patches, body)
		if desc, ok := body["description"].(string); ok {
			description = desc
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"uuid": svcUUID})
	})
	mux.HandleFunc("DELETE /api/v1/services/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deleted = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(acctest.WithVersionEndpoint(mux))
	defer srv.Close()

	base := acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service" "test" {
  project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
  environment_uuid = "` + envUUID + `"
  type             = "plausible"
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: base + "\n}\n",
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("coolify_service.test", "environment_uuid", envUUID),
					resource.TestCheckNoResourceAttr("coolify_service.test", "environment_name"),
				),
			},
			{
				Config: base + "\n  description = \"renamed\"\n}\n",
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("coolify_service.test", plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				Config: acctest.ProviderBlockForURL(srv.URL) + `
resource "coolify_service" "test" {
  project_uuid     = "aaaa0001-0001-4000-8000-000000000001"
  server_uuid      = "bbbb0001-0001-4000-8000-000000000001"
  environment_uuid = "eeee0002-0002-4000-8000-000000000002"
  type             = "plausible"
  description      = "renamed"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("coolify_service.test", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	if len(posts) < 1 {
		t.Fatal("expected a create POST")
	}
	if _, ok := posts[0]["environment_name"]; ok {
		t.Fatalf("create sent environment_name: %#v", posts[0]["environment_name"])
	}
	if posts[0]["environment_uuid"] != envUUID {
		t.Fatalf("environment_uuid = %#v", posts[0]["environment_uuid"])
	}
	if len(patches) == 0 {
		t.Fatal("expected an update PATCH")
	}
	if _, ok := patches[0]["environment_uuid"]; ok {
		t.Fatalf("update sent environment_uuid: %#v", patches[0]["environment_uuid"])
	}
}

func decodeBody(r *http.Request) (map[string]any, error) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	return body, nil
}

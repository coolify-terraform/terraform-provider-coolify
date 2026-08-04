package application

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestDockerComposeDomainsDescription_DocumentsOrderingAndVersions locks the
// operator-facing docs for issue #653. Coolify rejects docker_compose_domains
// until docker_compose_raw is set; that guard is present on every Coolify
// version this provider supports (minCoolifyVersion 4.1.0 through 4.2.0+).
func TestDockerComposeDomainsDescription_DocumentsOrderingAndVersions(t *testing.T) {
	t.Parallel()

	required := []string{
		"dockercompose",
		"jsonencode",
		"name",
		"domain",
		"docker_compose_raw",
		"deployment",
		"two-stage",
		"instant_deploy",
		"v4.1.0",
		"coolify_service",
		"load-compose",
		"normalizes", // #652: array write vs object read
	}
	for _, needle := range required {
		if !strings.Contains(dockerComposeDomainsDescription, needle) {
			t.Errorf("dockerComposeDomainsDescription missing %q", needle)
		}
	}

	// Must not claim a version floor higher than minCoolifyVersion for this
	// constraint: the empty-raw guard exists on 4.1.0, 4.1.1, 4.1.2, and 4.2.0.
	if strings.Contains(dockerComposeDomainsDescription, "Requires Coolify >= v4.2") {
		t.Error("ordering constraint applies on all supported versions; do not mark it v4.2.0-only")
	}
}

func TestAnnotateDockerComposeDomainsError(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		if got := annotateDockerComposeDomainsError(nil); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("unrelated", func(t *testing.T) {
		t.Parallel()
		err := errors.New(`unexpected status 422: {"message":"Validation failed.","errors":{"name":"required"}}`)
		if got := annotateDockerComposeDomainsError(err); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("coolify empty raw", func(t *testing.T) {
		t.Parallel()
		err := errors.New(`unexpected status 422: {"message":"Validation failed.","errors":{"docker_compose_domains":"` +
			coolifyDockerComposeDomainsNeedRaw + `. Reload the compose file from the git repository first."}}`)
		got := annotateDockerComposeDomainsError(err)
		if got == "" {
			t.Fatal("expected annotation, got empty")
		}
		for _, needle := range []string{
			"docker_compose_raw",
			"docker_compose_domains",
			"v4.1.0",
			"second apply",
			"deploy once",
		} {
			if !strings.Contains(got, needle) {
				t.Errorf("annotation missing %q: %s", needle, got)
			}
		}
	})
}

// TestCoolifyDockerComposeDomainsNeedRawMatchesUpstream locks the substring we
// match against Coolify controller error text so a typo does not silently drop
// the operator guidance.
func TestCoolifyDockerComposeDomainsNeedRawMatchesUpstream(t *testing.T) {
	t.Parallel()
	// Exact text from ApplicationsController on Coolify v4.1.0–v4.2.0.
	const upstream = "Cannot set docker_compose_domains without docker_compose_raw"
	if coolifyDockerComposeDomainsNeedRaw != upstream {
		t.Errorf("coolifyDockerComposeDomainsNeedRaw = %q, want %q", coolifyDockerComposeDomainsNeedRaw, upstream)
	}
}

// Coolify ApplicationsController stores array write form as an object map:
// put(name, ['domain' => domain]) then json_encode. #652.
func TestNormalizeDockerComposeDomains_ObjectToArray(t *testing.T) {
	t.Parallel()
	// Storage form returned by GET (string column).
	api := `{"grafana":{"domain":"http://grafana.example.com"},"web":{"domain":"https://app.example.com"}}`
	got := normalizeDockerComposeDomains(api)
	// Sorted by name for stable plans.
	want := `[{"name":"grafana","domain":"http://grafana.example.com"},{"name":"web","domain":"https://app.example.com"}]`
	if got != want {
		t.Errorf("normalize = %s, want %s", got, want)
	}
}

func TestNormalizeDockerComposeDomains_ArrayStable(t *testing.T) {
	t.Parallel()
	// Write form; order may differ from sorted canonical.
	in := `[{"name":"web","domain":"https://app.example.com"},{"name":"grafana","domain":"http://grafana.example.com"}]`
	got := normalizeDockerComposeDomains(in)
	want := `[{"name":"grafana","domain":"http://grafana.example.com"},{"name":"web","domain":"https://app.example.com"}]`
	if got != want {
		t.Errorf("normalize = %s, want %s", got, want)
	}
}

func TestNormalizeDockerComposeDomains_Empty(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"", "null", "[]", "{}", "  "} {
		if got := normalizeDockerComposeDomains(in); got != "" {
			t.Errorf("normalize(%q) = %q, want empty", in, got)
		}
	}
}

func TestDockerComposeDomainsEquivalent_ArrayVsObject(t *testing.T) {
	t.Parallel()
	arr := `[{"name":"grafana","domain":"http://grafana.example.com"}]`
	obj := `{"grafana":{"domain":"http://grafana.example.com"}}`
	if !dockerComposeDomainsEquivalent(arr, obj) {
		t.Fatal("array write form and object storage form should be equivalent")
	}
	if dockerComposeDomainsEquivalent(arr, `{"other":{"domain":"http://x"}}`) {
		t.Fatal("different service names should not be equivalent")
	}
}

func TestWireDockerComposeDomains_IsJSONArrayNotString(t *testing.T) {
	t.Parallel()
	// User config after jsonencode (array form) or after read of object form.
	raw := wireDockerComposeDomains(`{"grafana":{"domain":"http://grafana.example.com"}}`)
	if raw == nil {
		t.Fatal("expected non-nil raw")
	}
	// Must be a JSON array value, not a quoted string (Coolify 'array' rule).
	if raw[0] != '[' {
		t.Fatalf("wire must start with '[' for JSON array, got %s", raw)
	}
	var items []composeDomainItem
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("unmarshal wire: %v", err)
	}
	if len(items) != 1 || items[0].Name != "grafana" {
		t.Fatalf("items = %+v", items)
	}

	// When embedded in UpdateApplicationInput, body must not double-encode.
	type body struct {
		DockerComposeDomains json.RawMessage `json:"docker_compose_domains,omitempty"`
	}
	b, err := json.Marshal(body{DockerComposeDomains: raw})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["docker_compose_domains"][0] != '[' {
		t.Fatalf("marshaled field must be array, got %s", b)
	}
}

func TestResolveDockerComposeDomains_PreservesEquivalentUserForm(t *testing.T) {
	t.Parallel()
	// Terraform jsonencode is compact array form.
	user := `[{"name":"grafana","domain":"http://grafana.example.com"}]`
	// API returns storage object as a JSON string value.
	api := `{"grafana":{"domain":"http://grafana.example.com"}}`
	dst := types.StringValue(user)
	resolveDockerComposeDomains(&dst, api)
	if dst.ValueString() != user {
		t.Errorf("should preserve user form when equivalent: got %s", dst.ValueString())
	}
}

func TestResolveDockerComposeDomains_ImportFromAPI(t *testing.T) {
	t.Parallel()
	api := `{"web":{"domain":"https://app.example.com"}}`
	dst := types.StringNull()
	resolveDockerComposeDomains(&dst, api)
	want := `[{"name":"web","domain":"https://app.example.com"}]`
	if dst.ValueString() != want {
		t.Errorf("import normalize = %s, want %s", dst.ValueString(), want)
	}
}

func TestDockerComposeDomainsIfChanged_SkipsEquivalentShapes(t *testing.T) {
	t.Parallel()
	plan := types.StringValue(`[{"name":"web","domain":"https://a.example.com"}]`)
	state := types.StringValue(`{"web":{"domain":"https://a.example.com"}}`)
	if got := dockerComposeDomainsIfChanged(plan, state); got != nil {
		t.Errorf("expected nil (no PATCH) for equivalent shapes, got %s", got)
	}
	state2 := types.StringValue(`{"web":{"domain":"https://b.example.com"}}`)
	if got := dockerComposeDomainsIfChanged(plan, state2); got == nil {
		t.Fatal("expected wire payload when domain URL changed")
	}
}

func TestDockerComposeDomainsFieldChanged(t *testing.T) {
	t.Parallel()
	a := types.StringValue(`[{"name":"web","domain":"https://a.example.com"}]`)
	b := types.StringValue(`{"web":{"domain":"https://a.example.com"}}`)
	if dockerComposeDomainsFieldChanged(&a, &b) {
		t.Error("equivalent shapes should not count as changed")
	}
	c := types.StringValue(`{"web":{"domain":"https://b.example.com"}}`)
	if !dockerComposeDomainsFieldChanged(&a, &c) {
		t.Error("different domain should count as changed")
	}
}

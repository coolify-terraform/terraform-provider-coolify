package spectest

import (
	"sort"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
)

// TestWriteCoverage_ApplicationEnv ensures application env create/update
// allowed_fields appear on the client write payload type (#623 Phase A).
func TestWriteCoverage_ApplicationEnv(t *testing.T) {
	t.Parallel()
	c := loadContract(t)

	endpoints := []string{
		"ApplicationsController::create_env",
		"ApplicationsController::update_env_by_uuid",
	}
	tags := client.EnvVarWriteJSONTags("applications")
	if len(tags) == 0 {
		t.Fatal("EnvVarWriteJSONTags(applications) returned empty set")
	}
	requireValidSkips(t, appEnvWriteSkips)

	for _, name := range endpoints {
		ep, ok := c.Endpoints[name]
		if !ok {
			t.Fatalf("endpoint %s not found in contract", name)
		}
		if len(ep.AllowedFields) == 0 {
			t.Fatalf("endpoint %s has empty allowed_fields", name)
		}
		var missing []string
		for _, field := range ep.AllowedFields {
			if isSkipped(appEnvWriteSkips, field) {
				continue
			}
			if _, ok := tags[field]; !ok {
				missing = append(missing, field)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("%s allowed_fields missing from application env write input:\n  %s",
				name, strings.Join(missing, "\n  "))
		}
	}
}

// TestWriteCoverage_ServiceEnvSanity ensures service write payloads never
// claim is_runtime / is_buildtime (parent-scoped allow lists).
func TestWriteCoverage_ServiceEnvSanity(t *testing.T) {
	t.Parallel()
	tags := client.EnvVarWriteJSONTags("services")
	for _, forbidden := range []string{"is_runtime", "is_buildtime"} {
		if _, ok := tags[forbidden]; ok {
			t.Errorf("service env write input must not include %s", forbidden)
		}
	}
	for _, required := range []string{"key", "value", "is_preview", "is_literal", "is_multiline", "comment"} {
		if _, ok := tags[required]; !ok {
			t.Errorf("service env write input missing %s", required)
		}
	}
}

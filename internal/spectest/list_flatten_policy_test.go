package spectest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// resourceListPolicy is the #818 inventory: every Terraform List / ListNested
// attribute on a resource must have an explicit GET-order policy. Echoing the
// POST body on GET hid the service urls bug.
//
// If this test fails, add the new attribute here and add a flatten test that
// returns the API list in a different order than HCL (or document why GET
// cannot reorder it).
var resourceListPolicy = map[string]string{
	"internal/service/service/resource.go:urls":                     "TestFlattenServiceURLs_PreservesConfigOrder",
	"internal/service/application/common_schema.go:noindex_domains": "TestFlattenNoindexDomains_PreservesConfigOrder",
	"internal/service/hetzner/resource.go:hetzner_firewall_ids":     "create-only; GET omitted; flatten does not overwrite",
	"internal/service/hetzner/resource.go:hetzner_network_ids":      "create-only; GET omitted; flatten does not overwrite",
}

var listAttrDecl = regexp.MustCompile(`(?m)^\s+"([^"]+)": schema\.(ListAttribute|ListNestedAttribute)\{`)

func TestResourceListAttributes_HaveOrderPolicy(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "service")
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Dir(filepath.Dir(absRoot))
	found := map[string]bool{}
	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, "data_source") {
			return nil
		}
		if base != "resource.go" && !strings.HasPrefix(base, "resource_") && base != "common_schema.go" && base != "common.go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		for _, m := range listAttrDecl.FindAllSubmatch(data, -1) {
			key := rel + ":" + string(m[1])
			found[key] = true
			if _, ok := resourceListPolicy[key]; !ok {
				t.Errorf("resource list %s has no GET-order policy; add it to resourceListPolicy in list_flatten_policy_test.go", key)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for key := range resourceListPolicy {
		if !found[key] {
			t.Errorf("resourceListPolicy has stale key %s (attribute moved or renamed)", key)
		}
	}
}

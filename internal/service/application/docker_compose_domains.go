package application

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// composeDomainItem is Coolify's write shape for docker_compose_domains:
// ApplicationsController validates array of {name, domain} then stores a map
// keyed by service name (see ApplicationsController create/update).
type composeDomainItem struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// normalizeDockerComposeDomains converts Coolify GET/storage form (object map
// JSON string) or write form (array JSON) into the canonical write-shape JSON
// array string, sorted by service name for stable Terraform plans (#652).
//
// Coolify write (validation):
//
//	[{"name":"web","domain":"https://app.example.com"}]
//
// Coolify store / GET (string column, no cast):
//
//	{"web":{"domain":"https://app.example.com"}}
func normalizeDockerComposeDomains(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return ""
	}
	items, ok := parseDockerComposeDomains(raw)
	if !ok {
		return raw
	}
	if len(items) == 0 {
		return ""
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].Domain < items[j].Domain
	})
	b, err := json.Marshal(items)
	if err != nil {
		return raw
	}
	return string(b)
}

// parseDockerComposeDomains accepts Coolify's write array form, storage object
// map form, or a JSON-encoded string of either.
func parseDockerComposeDomains(raw string) ([]composeDomainItem, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil, true
	}

	// Write form: [{"name":"...","domain":"..."}]
	if strings.HasPrefix(raw, "[") {
		var arr []composeDomainItem
		if err := json.Unmarshal([]byte(raw), &arr); err != nil {
			return nil, false
		}
		return arr, true
	}

	// Storage form: {"svc":{"domain":"..."}}
	if strings.HasPrefix(raw, "{") {
		var obj map[string]struct {
			Domain string `json:"domain"`
		}
		if err := json.Unmarshal([]byte(raw), &obj); err != nil {
			return nil, false
		}
		items := make([]composeDomainItem, 0, len(obj))
		for name, v := range obj {
			items = append(items, composeDomainItem{Name: name, Domain: v.Domain})
		}
		return items, true
	}

	// Double-encoded JSON string (defensive).
	var inner string
	if err := json.Unmarshal([]byte(raw), &inner); err == nil && inner != raw {
		return parseDockerComposeDomains(inner)
	}
	return nil, false
}

// dockerComposeDomainsEquivalent reports whether two JSON strings describe the
// same Coolify domain mapping (array write shape or object storage shape).
func dockerComposeDomainsEquivalent(a, b string) bool {
	return normalizeDockerComposeDomains(a) == normalizeDockerComposeDomains(b)
}

// wireDockerComposeDomains returns Coolify write-shape JSON (a real JSON array
// value for the request body). Empty / unparseable-to-empty yields nil so the
// field is omitted via omitempty on json.RawMessage.
func wireDockerComposeDomains(raw string) json.RawMessage {
	n := normalizeDockerComposeDomains(raw)
	if n == "" {
		return nil
	}
	return json.RawMessage(n)
}

// resolveDockerComposeDomains sets state from the API while preserving the
// user's config string when it is semantically equal (avoids perpetual diffs
// from jsonencode spacing vs our canonical marshal).
func resolveDockerComposeDomains(dst *types.String, api string) {
	if dst == nil {
		return
	}
	if dst.IsNull() || dst.IsUnknown() {
		n := normalizeDockerComposeDomains(api)
		if n != "" {
			*dst = types.StringValue(n)
		}
		return
	}
	if dockerComposeDomainsEquivalent(dst.ValueString(), api) {
		return
	}
	n := normalizeDockerComposeDomains(api)
	if n == "" {
		*dst = types.StringNull()
		return
	}
	*dst = types.StringValue(n)
}

// dockerComposeDomainsFieldChanged is like stringFieldChanged but compares
// Coolify domain mappings semantically (array vs object form).
func dockerComposeDomainsFieldChanged(plan, state *types.String) bool {
	if plan == nil || state == nil {
		return false
	}
	if plan.IsNull() && state.IsNull() {
		return false
	}
	if plan.IsNull() || state.IsNull() {
		return true
	}
	return !dockerComposeDomainsEquivalent(plan.ValueString(), state.ValueString())
}

// dockerComposeDomainsIfChanged returns write-shape JSON for PATCH when plan
// and state are not semantically equal; nil when unchanged (omit from body).
func dockerComposeDomainsIfChanged(plan, state types.String) json.RawMessage {
	if plan.Equal(state) {
		return nil
	}
	if !plan.IsNull() && !plan.IsUnknown() && !state.IsNull() && !state.IsUnknown() &&
		dockerComposeDomainsEquivalent(plan.ValueString(), state.ValueString()) {
		return nil
	}
	if plan.IsNull() || plan.IsUnknown() {
		return nil
	}
	return wireDockerComposeDomains(plan.ValueString())
}

package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

const dockerComposeDomainsNotArray = `docker_compose_domains must be a JSON array of {name, domain, redirect} objects, not an object map. Write jsonencode([{ name = "web", domain = "https://pr.example.com" }]). Coolify GET uses {"web":{"domain":"..."}}.`

const dockerComposeDomainsRequired = "docker_compose_domains items require non-empty string name and domain"

type dockerComposeDomainsValidator struct{}

// DockerComposeDomains returns a string validator for Coolify preview
// docker_compose_domains. Null, unknown, and empty values are allowed.
// Non-empty values must be a JSON array of objects, each with non-empty
// string name and domain. Extra keys are not allowed. Optional redirect
// must be www, non-www, or both when present and non-null.
func DockerComposeDomains() validator.String {
	return dockerComposeDomainsValidator{}
}

func (v dockerComposeDomainsValidator) Description(_ context.Context) string {
	return "empty string, or a JSON array of objects with non-empty string name and domain; only name, domain, and optional redirect (www, non-www, both) are allowed"
}

func (v dockerComposeDomainsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v dockerComposeDomainsValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := strings.TrimSpace(req.ConfigValue.ValueString())
	if val == "" {
		return
	}

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(val), &items); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Docker Compose Domains", dockerComposeDomainsNotArray)
		return
	}
	for _, item := range items {
		if detail := previewComposeDomainItemError(item); detail != "" {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid Docker Compose Domains", detail)
			return
		}
	}
}

func previewComposeDomainItemError(item json.RawMessage) string {
	var obj map[string]any
	if err := json.Unmarshal(item, &obj); err != nil {
		return dockerComposeDomainsRequired
	}
	var unknown []string
	for key := range obj {
		if !allowedPreviewComposeDomainKey(key) {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Sprintf("docker_compose_domains has unknown field %q; allowed fields are name, domain, redirect", unknown[0])
	}
	name, nameOK := obj["name"].(string)
	domain, domainOK := obj["domain"].(string)
	if !nameOK || strings.TrimSpace(name) == "" || !domainOK || strings.TrimSpace(domain) == "" {
		return dockerComposeDomainsRequired
	}
	return previewComposeRedirectError(obj)
}

func allowedPreviewComposeDomainKey(key string) bool {
	return key == "name" || key == "domain" || key == "redirect"
}

func previewComposeRedirectError(obj map[string]any) string {
	redirect, ok := obj["redirect"]
	if !ok || redirect == nil {
		return ""
	}
	s, isStr := redirect.(string)
	if isStr && (s == "" || s == "www" || s == "non-www" || s == "both") {
		return ""
	}
	if isStr {
		return fmt.Sprintf("redirect must be www, non-www, or both, got %q", s)
	}
	return fmt.Sprintf("redirect must be www, non-www, or both, got %q", fmt.Sprint(redirect))
}

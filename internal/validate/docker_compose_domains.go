package validate

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

const dockerComposeDomainsDetail = "docker_compose_domains must be a JSON array of objects with non-empty string name and domain; only name, domain, and redirect fields are supported"

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
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Docker Compose Domains", dockerComposeDomainsDetail)
		return
	}
	for _, item := range items {
		if !validPreviewComposeDomainItem(item) {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid Docker Compose Domains", dockerComposeDomainsDetail)
			return
		}
	}
}

func validPreviewComposeDomainItem(item json.RawMessage) bool {
	var obj map[string]any
	if err := json.Unmarshal(item, &obj); err != nil {
		return false
	}
	for key := range obj {
		if !allowedPreviewComposeDomainKey(key) {
			return false
		}
	}
	name, nameOK := obj["name"].(string)
	domain, domainOK := obj["domain"].(string)
	if !nameOK || strings.TrimSpace(name) == "" || !domainOK || strings.TrimSpace(domain) == "" {
		return false
	}
	return validPreviewComposeRedirect(obj)
}

func allowedPreviewComposeDomainKey(key string) bool {
	return key == "name" || key == "domain" || key == "redirect"
}

func validPreviewComposeRedirect(obj map[string]any) bool {
	redirect, ok := obj["redirect"]
	if !ok || redirect == nil {
		return true
	}
	s, isStr := redirect.(string)
	return isStr && (s == "" || s == "www" || s == "non-www" || s == "both")
}

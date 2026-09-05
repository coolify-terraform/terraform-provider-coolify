package validate

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

const dockerComposeDomainsDetail = "docker_compose_domains must be a JSON array of objects with non-empty string name and domain"

type dockerComposeDomainsValidator struct{}

// DockerComposeDomains returns a string validator for Coolify preview
// docker_compose_domains. Null, unknown, and empty values are allowed.
// Non-empty values must be a JSON array of objects, each with non-empty
// string name and domain. Extra keys such as redirect are allowed.
func DockerComposeDomains() validator.String {
	return dockerComposeDomainsValidator{}
}

func (v dockerComposeDomainsValidator) Description(_ context.Context) string {
	return "empty string, or a JSON array of objects with non-empty string name and domain"
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
		var obj map[string]any
		if err := json.Unmarshal(item, &obj); err != nil {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid Docker Compose Domains", dockerComposeDomainsDetail)
			return
		}
		name, nameOK := obj["name"].(string)
		domain, domainOK := obj["domain"].(string)
		if !nameOK || strings.TrimSpace(name) == "" || !domainOK || strings.TrimSpace(domain) == "" {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid Docker Compose Domains", dockerComposeDomainsDetail)
			return
		}
	}
}

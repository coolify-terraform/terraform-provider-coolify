package validate

import (
	"context"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type domainsValidator struct{}

// Domains returns a string validator for Coolify application domains.
// Empty string is allowed (clears FQDN on update; create with empty +
// autogenerate_domain=false leaves no public domain). Non-empty values must
// be comma-separated http(s) URLs with non-empty hosts.
func Domains() validator.String {
	return domainsValidator{}
}

func (v domainsValidator) Description(_ context.Context) string {
	return "empty string, or comma-separated http:// or https:// URLs with non-empty hosts"
}

func (v domainsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v domainsValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := strings.TrimSpace(req.ConfigValue.ValueString())
	if val == "" {
		return
	}
	for _, part := range strings.Split(val, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		u, err := url.Parse(part)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid URL",
				"must be empty, or comma-separated http:// or https:// URLs with non-empty hosts, got: "+part,
			)
			return
		}
	}
}

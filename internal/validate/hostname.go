package validate

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// hostnameLabel is one RFC 1123 DNS label: letters, digits, hyphens;
// not starting or ending with a hyphen.
var hostnameLabel = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)

// Hostname returns a string validator for Coolify ValidHostname (RFC 1123).
// Null, unknown, and empty values are allowed (the API field is nullable).
func Hostname() validator.String {
	return hostnameValidator{}
}

type hostnameValidator struct{}

func (v hostnameValidator) Description(_ context.Context) string {
	return "RFC 1123 hostname (letters, digits, hyphens, dots; max 253 characters)"
}

func (v hostnameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v hostnameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := strings.TrimSpace(req.ConfigValue.ValueString())
	if val == "" {
		return
	}
	if len(val) > 253 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid hostname",
			"must be at most 253 characters",
		)
		return
	}
	for _, r := range val {
		if r < 32 || r == 127 {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid hostname",
				"must not contain control characters",
			)
			return
		}
	}
	for _, label := range strings.Split(val, ".") {
		if label == "" || !hostnameLabel.MatchString(label) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid hostname",
				"must be an RFC 1123 hostname (letters, digits, hyphens, and dots; labels cannot start or end with a hyphen)",
			)
			return
		}
	}
}

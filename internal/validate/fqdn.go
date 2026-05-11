package validate

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type fqdnValidator struct{}

// FQDN returns a string validator that checks if the value is a valid HTTP(S) URL
// with a non-empty host.
func FQDN() validator.String {
	return fqdnValidator{}
}

func (v fqdnValidator) Description(_ context.Context) string {
	return "must be a valid URL starting with http:// or https:// with a non-empty host"
}

func (v fqdnValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v fqdnValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	u, err := url.Parse(val)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid URL",
			"must be a valid URL starting with http:// or https:// with a non-empty host, got: "+val,
		)
	}
}

package validate

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type domainsValidator struct{}

// Domains returns a string validator that checks if the value is a valid HTTP(S) URL
// with a non-empty host.
func Domains() validator.String {
	return domainsValidator{}
}

func (v domainsValidator) Description(_ context.Context) string {
	return "must be a valid URL starting with http:// or https:// with a non-empty host"
}

func (v domainsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v domainsValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
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

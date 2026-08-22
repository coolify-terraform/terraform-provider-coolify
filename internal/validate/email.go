package validate

import (
	"context"
	"net/mail"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Email returns a string validator for a single RFC 5322 mailbox address.
// Null, unknown, and empty values are allowed (Coolify fields are nullable).
func Email() validator.String {
	return emailValidator{}
}

type emailValidator struct{}

func (v emailValidator) Description(_ context.Context) string {
	return "valid email address"
}

func (v emailValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v emailValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := strings.TrimSpace(req.ConfigValue.ValueString())
	if val == "" {
		return
	}
	addr, err := mail.ParseAddress(val)
	if err != nil || addr.Address == "" || !strings.Contains(addr.Address, "@") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid email address",
			"must be a single email address (for example alerts@example.com)",
		)
		return
	}
	if addr.Address != val && addr.Name != "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid email address",
			"must be a bare email address, not a display-name form",
		)
	}
}

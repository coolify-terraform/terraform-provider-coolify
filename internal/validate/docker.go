package validate

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// shellMetachars matches characters that could enable shell injection
// when the value is interpolated into a command line.
var shellMetachars = regexp.MustCompile("[;|&`$(){}]")

// NoShellMetachars returns a string validator that rejects values
// containing shell metacharacters (; | & ` $ ( ) { }).
func NoShellMetachars() validator.String {
	return noShellMetacharsValidator{}
}

type noShellMetacharsValidator struct{}

func (v noShellMetacharsValidator) Description(_ context.Context) string {
	return "must not contain shell metacharacters (; | & ` $ ( ) { })"
}

func (v noShellMetacharsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v noShellMetacharsValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if loc := shellMetachars.FindStringIndex(value); loc != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Shell Metacharacter Detected",
			v.Description(ctx))
	}
}

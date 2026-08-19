package validate_test

import (
	"context"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestHostname(t *testing.T) {
	t.Parallel()
	v := validate.Hostname()
	ctx := context.Background()

	ok := []string{"mail.example.com", "smtp.example.com", "mail", "EHLO.Example.COM", "123.example.com", "10.0.0.1", ""}
	for _, in := range ok {
		req := validator.StringRequest{ConfigValue: types.StringValue(in)}
		var resp validator.StringResponse
		v.ValidateString(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("Hostname(%q) unexpected error: %v", in, resp.Diagnostics)
		}
	}

	nullReq := validator.StringRequest{ConfigValue: types.StringNull()}
	var nullResp validator.StringResponse
	v.ValidateString(ctx, nullReq, &nullResp)
	if nullResp.Diagnostics.HasError() {
		t.Errorf("Hostname(null) unexpected error: %v", nullResp.Diagnostics)
	}

	tooLong := strings.Repeat("a", 254)
	bad := []string{"-bad.example.com", "bad-.example.com", "has space.com", "mail_host", tooLong}
	for _, in := range bad {
		req := validator.StringRequest{ConfigValue: types.StringValue(in)}
		var resp validator.StringResponse
		v.ValidateString(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("Hostname(%q) want error", in)
		}
	}
}

package validate_test

import (
	"context"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEmail(t *testing.T) {
	t.Parallel()
	v := validate.Email()
	ctx := context.Background()

	ok := []string{"alerts@example.com", "ops+pager@mail.example.com", ""}
	for _, in := range ok {
		req := validator.StringRequest{ConfigValue: types.StringValue(in)}
		var resp validator.StringResponse
		v.ValidateString(ctx, req, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("Email(%q) unexpected error: %v", in, resp.Diagnostics)
		}
	}

	nullReq := validator.StringRequest{ConfigValue: types.StringNull()}
	var nullResp validator.StringResponse
	v.ValidateString(ctx, nullReq, &nullResp)
	if nullResp.Diagnostics.HasError() {
		t.Errorf("Email(null) unexpected error: %v", nullResp.Diagnostics)
	}

	bad := []string{"not-an-email", "alerts@", "@example.com", "a b@example.com", "Alerts <alerts@example.com>"}
	for _, in := range bad {
		req := validator.StringRequest{ConfigValue: types.StringValue(in)}
		var resp validator.StringResponse
		v.ValidateString(ctx, req, &resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("Email(%q) want error", in)
		}
	}
}

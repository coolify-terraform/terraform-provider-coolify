package validate_test

import (
	"context"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCoolifyFrequency(t *testing.T) {
	t.Parallel()
	v := validate.CoolifyFrequency()
	valid := []string{
		"0 2 * * *",
		"*/5 * * * *",
		"@daily",
		"@hourly",
		"daily",
		"hourly",
		"weekly",
		"monthly",
		"yearly",
		"every_minute",
	}
	for _, s := range valid {
		s := s
		t.Run("ok_"+s, func(t *testing.T) {
			t.Parallel()
			req := validator.StringRequest{ConfigValue: types.StringValue(s)}
			var resp validator.StringResponse
			v.ValidateString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("%q should be valid: %v", s, resp.Diagnostics)
			}
		})
	}
	invalid := []string{"not-a-cron", "@secondly", "every minute"}
	for _, s := range invalid {
		s := s
		t.Run("bad_"+s, func(t *testing.T) {
			t.Parallel()
			req := validator.StringRequest{ConfigValue: types.StringValue(s)}
			var resp validator.StringResponse
			v.ValidateString(context.Background(), req, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatalf("%q should be invalid", s)
			}
		})
	}
}

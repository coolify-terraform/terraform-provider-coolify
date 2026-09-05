package validate_test

import (
	"context"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDockerComposeDomains_Valid(t *testing.T) {
	t.Parallel()
	valid := []string{
		"",
		" ",
		"[]",
		`[{"name":"web","domain":"https://pr.example.com"}]`,
		`[{"name":"web","domain":"https://pr.example.com","redirect":"www"}]`,
		`[{"name":"web","domain":"https://pr.example.com","redirect":"non-www"}]`,
		`[{"name":"web","domain":"https://pr.example.com","redirect":"both"}]`,
		`[{"name":"web","domain":"https://pr.example.com","redirect":null}]`,
		`[{"name":"web","domain":"https://pr.example.com","redirect":""}]`,
		`[{"name":"web","domain":"https://a.example.com"},{"name":"api","domain":"https://b.example.com"}]`,
	}
	v := validate.DockerComposeDomains()
	for _, s := range valid {
		resp := validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: types.StringValue(s),
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("DockerComposeDomains(%q) should be valid, got error: %s", s, resp.Diagnostics.Errors()[0].Detail())
		}
	}
}

func TestDockerComposeDomains_Invalid(t *testing.T) {
	t.Parallel()
	invalid := []string{
		"not-json",
		`{"web":{"domain":"https://pr.example.com"}}`,
		`"web"`,
		"[1]",
		"[{}]",
		`[{"name":"web"}]`,
		`[{"domain":"https://pr.example.com"}]`,
		`[{"name":"","domain":"https://pr.example.com"}]`,
		`[{"name":"web","domain":""}]`,
		`[{"name":1,"domain":"https://pr.example.com"}]`,
		`[{"name":"web","domain":1}]`,
		`[{"name":"web","domain":"https://pr.example.com","port":80}]`,
		`[{"name":"web","domain":"https://pr.example.com","redirect":"always"}]`,
	}
	v := validate.DockerComposeDomains()
	for _, s := range invalid {
		resp := validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: types.StringValue(s),
		}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("DockerComposeDomains(%q) should be invalid, got no error", s)
		}
	}
}

func TestDockerComposeDomains_NullAndUnknown(t *testing.T) {
	t.Parallel()
	v := validate.DockerComposeDomains()

	nullResp := validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringNull(),
	}, &nullResp)
	if nullResp.Diagnostics.HasError() {
		t.Error("DockerComposeDomains(null) should pass, got error")
	}

	unknownResp := validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringUnknown(),
	}, &unknownResp)
	if unknownResp.Diagnostics.HasError() {
		t.Error("DockerComposeDomains(unknown) should pass, got error")
	}
}

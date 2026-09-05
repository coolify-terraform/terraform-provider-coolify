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
	const notArray = `docker_compose_domains must be a JSON array of {name, domain, redirect} objects, not an object map. Write jsonencode([{ name = "web", domain = "https://pr.example.com" }]). Coolify GET uses {"web":{"domain":"..."}}.`
	const required = "docker_compose_domains items require non-empty string name and domain"
	const extraPort = `docker_compose_domains has unknown field "port"; allowed fields are name, domain, redirect`
	const badRedirect = `redirect must be www, non-www, or both, got "always"`
	tests := []struct {
		in   string
		want string
	}{
		{"not-json", notArray},
		{`{"web":{"domain":"https://pr.example.com"}}`, notArray},
		{`"web"`, notArray},
		{"[1]", required},
		{"[{}]", required},
		{`[{"name":"web"}]`, required},
		{`[{"domain":"https://pr.example.com"}]`, required},
		{`[{"name":"","domain":"https://pr.example.com"}]`, required},
		{`[{"name":"web","domain":""}]`, required},
		{`[{"name":1,"domain":"https://pr.example.com"}]`, required},
		{`[{"name":"web","domain":1}]`, required},
		{`[{"name":"web","domain":"https://pr.example.com","port":80}]`, extraPort},
		{`[{"name":"web","domain":"https://pr.example.com","redirect":"always"}]`, badRedirect},
	}
	v := validate.DockerComposeDomains()
	for _, tt := range tests {
		resp := validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: types.StringValue(tt.in),
		}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("DockerComposeDomains(%q) should be invalid, got no error", tt.in)
			continue
		}
		got := resp.Diagnostics.Errors()[0].Detail()
		if got != tt.want {
			t.Errorf("DockerComposeDomains(%q) detail = %q, want %q", tt.in, got, tt.want)
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

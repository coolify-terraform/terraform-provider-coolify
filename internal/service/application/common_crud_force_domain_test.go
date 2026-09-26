package application

import (
	"encoding/json"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
)

// TestWithForceDomainOverride pins when a PATCH carries force_domain_override:
// only when the configuration asks for it and the request writes domains.
func TestWithForceDomainOverride(t *testing.T) {
	t.Parallel()
	domains := "https://shared.example.com"
	compose := json.RawMessage(`[{"name":"web","domain":"https://shared.example.com"}]`)

	cases := []struct {
		name  string
		input client.UpdateApplicationInput
		force bool
		want  bool
	}{
		{"not configured", client.UpdateApplicationInput{Domains: &domains}, false, false},
		{"configured, domains unchanged", client.UpdateApplicationInput{}, true, false},
		{"configured, domains written", client.UpdateApplicationInput{Domains: &domains}, true, true},
		{"configured, compose domains written", client.UpdateApplicationInput{DockerComposeDomains: compose}, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := tc.input
			withForceDomainOverride(&input, tc.force)
			got := input.ForceDomainOverride != nil && *input.ForceDomainOverride
			if got != tc.want {
				t.Errorf("force_domain_override sent = %v, want %v", got, tc.want)
			}
			if !tc.want && input.ForceDomainOverride != nil {
				t.Errorf("force_domain_override should be omitted, got %v", *input.ForceDomainOverride)
			}
		})
	}
}

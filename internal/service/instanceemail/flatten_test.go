package instanceemail

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlatten_NilAPI(t *testing.T) {
	t.Parallel()
	var m model
	flatten(nil, &m)
	if !m.ID.IsNull() && !m.ID.IsUnknown() && m.ID.ValueString() != "" {
		t.Fatalf("flatten(nil) must not set id, got %#v", m.ID)
	}
	flatten(&client.InstanceEmailSettings{SMTPEnabled: true}, nil)
}

func TestFlatten_SMTPEnabled(t *testing.T) {
	t.Parallel()
	var m model
	flatten(&client.InstanceEmailSettings{SMTPEnabled: true, SMTPHost: "smtp.example.com"}, &m)
	if m.ID.ValueString() != "current" {
		t.Fatalf("id = %q, want current", m.ID.ValueString())
	}
	if !m.SMTPEnabled.Equal(types.BoolValue(true)) {
		t.Fatalf("smtp_enabled = %#v, want true", m.SMTPEnabled)
	}
	if m.SMTPHost.ValueString() != "smtp.example.com" {
		t.Fatalf("smtp_host = %q", m.SMTPHost.ValueString())
	}
}

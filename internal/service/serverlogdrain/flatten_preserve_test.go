package serverlogdrain

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlatten_EmptyCustomConfigPreservesState(t *testing.T) {
	t.Parallel()
	m := model{CustomConfig: types.StringValue("secret-config")}
	flatten(&client.ServerLogDrains{CustomConfig: ""}, &m)
	if m.CustomConfig.IsNull() || m.CustomConfig.ValueString() != "secret-config" {
		t.Fatalf("custom config = %#v, want secret-config", m.CustomConfig)
	}
}

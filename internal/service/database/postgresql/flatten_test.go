package postgresql

import (
	"encoding/json"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenDatabase_InitScriptsFollowsAPI(t *testing.T) {
	t.Parallel()
	m := &postgresqlDatabaseResourceModel{}
	m.InitScripts = types.StringNull()
	raw := json.RawMessage(`[{"filename":"init.sql","content":"select 1"}]`)
	flattenDatabase(&client.Database{InitScripts: raw}, m)
	if m.InitScripts.ValueString() != string(raw) {
		t.Fatalf("init_scripts = %q, want API body", m.InitScripts.ValueString())
	}

	m.InitScripts = types.StringValue(string(raw))
	flattenDatabase(&client.Database{}, m)
	if !m.InitScripts.IsNull() {
		t.Fatalf("init_scripts = %#v, want null when API is empty", m.InitScripts)
	}
}

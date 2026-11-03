package integrationtoken

import (
	"encoding/json"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenIntegrationToken_PreservesCapabilityOrder(t *testing.T) {
	t.Parallel()
	model := integrationTokenModel{
		Capabilities: stringListValue([]string{"tunnel", "dns"}),
	}
	flattenIntegrationToken(&client.IntegrationToken{
		UUID:         "11111111-1111-4111-8111-111111111111",
		Name:         "cf",
		Provider:     "cloudflare",
		Capabilities: []string{"dns", "tunnel"},
	}, &model)
	got := stringList(t.Context(), model.Capabilities)
	if len(got) != 2 || got[0] != "tunnel" || got[1] != "dns" {
		t.Fatalf("capabilities = %#v, want configured order", got)
	}
	if model.Name.ValueString() != "cf" || model.Provider != types.StringValue("cloudflare") {
		t.Fatalf("name/provider = %s %s", model.Name, model.Provider)
	}
}

func TestUpdateInput_ClearsOptionalMetadata(t *testing.T) {
	t.Parallel()
	plan := integrationTokenModel{
		Name:              types.StringValue("vault"),
		Provider:          types.StringValue("vault"),
		Token:             types.StringValue("secret"),
		MetadataBaseURL:   types.StringValue("https://vault.example"),
		MetadataNamespace: types.StringNull(),
		MetadataTimeout:   types.Int64Null(),
	}
	state := plan
	state.MetadataNamespace = types.StringValue("prod")
	state.MetadataTimeout = types.Int64Value(30)

	in := updateInput(t.Context(), plan, state)
	if in.Token == nil || *in.Token != "secret" {
		t.Fatal("clearing connection settings must resend the token")
	}
	var meta map[string]any
	if err := json.Unmarshal(in.Metadata, &meta); err != nil {
		t.Fatal(err)
	}
	if meta["namespace"] != "" {
		t.Fatalf("namespace = %#v, want empty string", meta["namespace"])
	}
	if _, ok := meta["timeout"]; !ok || meta["timeout"] != nil {
		t.Fatalf("timeout = %#v, want JSON null", meta["timeout"])
	}
	if _, ok := meta["base_url"]; ok {
		t.Fatal("unchanged base_url must be omitted")
	}
}

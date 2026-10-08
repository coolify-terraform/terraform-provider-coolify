package privatekey

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenPrivateKey_EmptyDescription(t *testing.T) {
	t.Parallel()
	kept := privateKeyResourceModel{Description: types.StringValue("")}
	flattenPrivateKey(&client.PrivateKey{UUID: "u", Name: "n", Description: ""}, &kept)
	if kept.Description.IsNull() || kept.Description.ValueString() != "" {
		t.Fatalf("configured empty description = %#v, want empty string", kept.Description)
	}

	imported := privateKeyResourceModel{Description: types.StringNull()}
	flattenPrivateKey(&client.PrivateKey{UUID: "u", Name: "n", Description: ""}, &imported)
	if !imported.Description.IsNull() {
		t.Fatalf("null prior with empty API description = %#v, want null", imported.Description)
	}
}

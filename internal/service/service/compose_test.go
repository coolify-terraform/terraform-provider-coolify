package service

import (
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPreserveDockerComposeRaw(t *testing.T) {
	t.Parallel()
	user := "services:\n  web:\n    image: nginx\n"
	api := "services:\n  web:\n    image: nginx\n"
	if got := preserveDockerComposeRaw(types.StringValue(user), api); got.ValueString() != user {
		t.Fatalf("equal YAML stored %q, want user string", got.ValueString())
	}

	reformatted := "services:\n  web:\n    image: \"nginx\"\n"
	if got := preserveDockerComposeRaw(types.StringValue(user), reformatted); got.ValueString() != user {
		t.Fatalf("semantic match stored %q, want user string", got.ValueString())
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(user))
	if got := preserveDockerComposeRaw(types.StringValue(encoded), api); got.ValueString() != encoded {
		t.Fatalf("base64 form stored %q, want encoded user string", got.ValueString())
	}

	drift := "services:\n  web:\n    image: redis\n"
	if got := preserveDockerComposeRaw(types.StringValue(user), drift); got.ValueString() != drift {
		t.Fatalf("drift stored %q, want API string", got.ValueString())
	}

	if got := preserveDockerComposeRaw(types.StringNull(), api); got.ValueString() != api {
		t.Fatalf("import stored %q, want API string", got.ValueString())
	}

	if got := preserveDockerComposeRaw(types.StringValue(user), ""); got.ValueString() != user {
		t.Fatalf("empty API stored %q, want user string", got.ValueString())
	}
	if got := preserveDockerComposeRaw(types.StringUnknown(), ""); !got.IsNull() {
		t.Fatalf("unknown plus empty API = %#v, want null", got)
	}
}

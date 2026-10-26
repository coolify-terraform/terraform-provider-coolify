package githubapp

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenGitHubApp_EmptySecretsPreserveState(t *testing.T) {
	t.Parallel()
	m := gitHubAppResourceModel{
		ClientSecret:  types.StringValue("secret"),
		WebhookSecret: types.StringValue("hook"),
	}
	flattenGitHubApp(&client.GitHubApp{Name: "app", ClientID: "cid", ClientSecret: "", WebhookSecret: ""}, &m)
	if m.ClientSecret.ValueString() != "secret" {
		t.Fatalf("client_secret = %q", m.ClientSecret.ValueString())
	}
	if m.WebhookSecret.ValueString() != "hook" {
		t.Fatalf("webhook_secret = %q", m.WebhookSecret.ValueString())
	}
}

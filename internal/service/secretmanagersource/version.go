package secretmanagersource

import (
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func coolify46Ready(c *client.Client, diags *diag.Diagnostics) bool {
	if c == nil || c.SupportsCoolify46() {
		return true
	}
	ver := c.CoolifyVersion
	if ver == "" {
		ver = "unknown"
	}
	diags.AddError(
		"Secret manager sources need Coolify >= v4.6.0",
		fmt.Sprintf("This Coolify instance (%s) does not have the secret manager source API. Upgrade Coolify before using coolify_secret_manager_source.", ver),
	)
	return false
}

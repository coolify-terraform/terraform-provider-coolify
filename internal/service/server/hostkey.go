package server

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func sshHostKeyAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ssh_host_key": schema.StringAttribute{
			MarkdownDescription: "SSH host key to trust, as public key lines (`type base64`) or known_hosts lines. " +
				"Coolify does not return the raw key. Requires Coolify >= v4.7.0. On older instances the provider keeps the value in state and does not send it. " +
				"After an upgrade to v4.7.0, the next refresh sends a stored key that Coolify has not pinned yet. " +
				"Removing the value clears the trusted key so Coolify trusts the key the server sends on the next connection.",
			Optional:   true,
			Sensitive:  true,
			Validators: []validator.String{stringvalidator.LengthAtMost(16384)},
		},
		"ssh_host_key_fingerprints": schema.ListAttribute{
			MarkdownDescription: "Fingerprints of the trusted SSH host key. Null when Coolify has not stored one, including on instances older than v4.7.0.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"ssh_host_key_pending_fingerprints": schema.ListAttribute{
			MarkdownDescription: "Fingerprints of an SSH host key Coolify has seen and not trusted yet. Null when there is none, including on instances older than v4.7.0.",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func sshHostKeyForCreate(v types.String) *string {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return nil
	}
	s := v.ValueString()
	return &s
}

func warnUnsupportedSSHHostKey(c *client.Client, v types.String, diags *diag.Diagnostics) {
	if diags == nil || c == nil || c.SupportsCoolify47() {
		return
	}
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return
	}
	ver := c.CoolifyVersion
	if ver == "" {
		ver = "unknown"
	}
	diags.AddWarning(
		"Coolify version cannot write ssh_host_key",
		fmt.Sprintf("This Coolify instance (%s) does not accept ssh_host_key (added in Coolify v4.7.0). "+
			"The provider will keep the value in Terraform state but will not send it. "+
			"Upgrade to Coolify v4.7.0 or later, or remove ssh_host_key from configuration.", ver),
	)
}

func syncSSHHostKey(ctx context.Context, c *client.Client, uuid string, plan, state types.String, diags *diag.Diagnostics) error {
	warnUnsupportedSSHHostKey(c, plan, diags)
	if c == nil || !c.SupportsCoolify47() || sshHostKeyEqual(plan, state) {
		return nil
	}
	return c.ResetServerHostKey(ctx, uuid, sshHostKeyForCreate(plan))
}

func sshHostKeyEqual(plan, state types.String) bool {
	if plan.IsUnknown() || state.IsUnknown() {
		return true
	}
	return plan.ValueString() == state.ValueString()
}

// noteUnappliedSSHHostKey drops a configured key from refreshed state when
// Coolify 4.7 has no pinned fingerprints. The configuration still has the
// key, so the next plan updates and POST /host-key/reset runs. Calling this
// from Update would store null while the plan still has the key.
func noteUnappliedSSHHostKey(c *client.Client, model *serverResourceModel) {
	if c == nil || !c.SupportsCoolify47() || model == nil {
		return
	}
	if model.SSHHostKey.IsNull() || model.SSHHostKey.IsUnknown() || model.SSHHostKey.ValueString() == "" {
		return
	}
	if sshHostKeyFingerprintsPresent(model.SSHHostKeyFingerprints) {
		return
	}
	model.SSHHostKey = types.StringNull()
}

func sshHostKeyFingerprintsPresent(v types.List) bool {
	return !v.IsNull() && !v.IsUnknown() && len(v.Elements()) > 0
}

func applySSHHostKeyRead(srv *client.Server, model *serverResourceModel) {
	if model.SSHHostKey.IsUnknown() {
		model.SSHHostKey = types.StringNull()
	}
	var trusted, pending []string
	if srv != nil && srv.Settings != nil {
		trusted = srv.Settings.SSHHostKeyFingerprints
		pending = srv.Settings.SSHHostKeyPendingFingerprints
	}
	model.SSHHostKeyFingerprints = stringListOrNull(trusted)
	model.SSHHostKeyPendingFingerprints = stringListOrNull(pending)
}

func stringListOrNull(vals []string) types.List {
	if vals == nil {
		return types.ListNull(types.StringType)
	}
	l, diags := types.ListValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}
	return l
}

package s3storage

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func plannedStorageType(plan s3StorageResourceModel) string {
	if plan.Type.IsNull() || plan.Type.IsUnknown() || plan.Type.ValueString() == "" {
		return "s3"
	}
	return plan.Type.ValueString()
}

func stringMissing(v types.String) bool {
	return v.IsNull() || v.IsUnknown() || v.ValueString() == ""
}

func stringConfigured(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
}

func knownString(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func nullUnknownStorage(m *s3StorageResourceModel) {
	if m.Description.IsUnknown() {
		m.Description = types.StringNull()
	}
	if m.Bucket.IsUnknown() {
		m.Bucket = types.StringNull()
	}
	if m.Region.IsUnknown() {
		m.Region = types.StringNull()
	}
	if m.Secret.IsUnknown() {
		m.Secret = types.StringNull()
	}
	if m.Type.IsUnknown() {
		m.Type = types.StringNull()
	}
	if m.PrivateKeyUUID.IsUnknown() {
		m.PrivateKeyUUID = types.StringNull()
	}
	if m.SFTPHostKey.IsUnknown() {
		m.SFTPHostKey = types.StringNull()
	}
	if m.UsePathStyleEndpoint.IsUnknown() {
		m.UsePathStyleEndpoint = types.BoolNull()
	}
	if m.SkipTLSVerification.IsUnknown() {
		m.SkipTLSVerification = types.BoolNull()
	}
}

func boolConfigured(v types.Bool) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// clearPrivateKeyWhenOmitted plans null when configuration drops a stored
// SFTP key. UseStateForUnknown would keep the state value, so Terraform
// would never send the null Coolify needs to switch back to a password.
type clearPrivateKeyWhenOmitted struct{}

func (clearPrivateKeyWhenOmitted) Description(context.Context) string {
	return "Clears private_key_uuid when configuration omits a previously stored key."
}

func (m clearPrivateKeyWhenOmitted) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (clearPrivateKeyWhenOmitted) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	if req.ConfigValue.IsNull() && !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && req.StateValue.ValueString() != "" {
		resp.PlanValue = types.StringNull()
		return
	}
	stringplanmodifier.UseStateForUnknown().PlanModifyString(ctx, req, resp)
}

// rejectStoragePlan stops create and update before a request Coolify 4.7
// would reject. Forbidden S3-only fields are judged from configuration, not
// the plan: Coolify stores a region placeholder and path-style flags for
// sftp, ftps, and webdav, and UseStateForUnknown copies those into the plan.
func rejectStoragePlan(c *client.Client, plan, cfg s3StorageResourceModel) diag.Diagnostics {
	var d diag.Diagnostics
	typ := plannedStorageType(plan)
	if typ != "s3" && (c == nil || !c.SupportsCoolify47()) {
		ver := "unknown"
		if c != nil && c.CoolifyVersion != "" {
			ver = c.CoolifyVersion
		}
		d.AddAttributeError(
			path.Root("type"),
			"Coolify version cannot set storage type",
			fmt.Sprintf("This Coolify instance (%s) only accepts s3 storage. sftp, ftps, and webdav were added in v4.7.0.", ver),
		)
		return d
	}
	if typ == "s3" {
		if stringMissing(plan.Bucket) {
			d.AddAttributeError(path.Root("bucket"), "bucket is required for s3 storage", "Set bucket when type is s3 or omitted.")
		}
		if stringMissing(plan.Region) {
			d.AddAttributeError(path.Root("region"), "region is required for s3 storage", "Set region when type is s3 or omitted.")
		}
		if stringConfigured(cfg.PrivateKeyUUID) {
			d.AddAttributeError(path.Root("private_key_uuid"), "private_key_uuid is only for sftp", "Remove private_key_uuid, or set type to sftp.")
		}
		rejectMissingSecret(&d, plan, cfg, typ)
		return d
	}
	if stringConfigured(cfg.Bucket) {
		d.AddAttributeError(path.Root("bucket"), "bucket is only for s3 storage", fmt.Sprintf("Remove bucket when type is %s. Coolify rejects the field.", typ))
	}
	if stringConfigured(cfg.Region) {
		d.AddAttributeError(path.Root("region"), "region is only for s3 storage", fmt.Sprintf("Remove region when type is %s. Coolify rejects the field.", typ))
	}
	if boolConfigured(cfg.UsePathStyleEndpoint) {
		d.AddAttributeError(path.Root("use_path_style_endpoint"), "use_path_style_endpoint is only for s3 storage", fmt.Sprintf("Remove use_path_style_endpoint when type is %s.", typ))
	}
	if boolConfigured(cfg.SkipTLSVerification) {
		d.AddAttributeError(path.Root("skip_tls_verification"), "skip_tls_verification is only for s3 storage", fmt.Sprintf("Remove skip_tls_verification when type is %s.", typ))
	}
	if typ != "sftp" && stringConfigured(cfg.PrivateKeyUUID) {
		d.AddAttributeError(path.Root("private_key_uuid"), "private_key_uuid is only for sftp", fmt.Sprintf("Remove private_key_uuid when type is %s.", typ))
	}
	rejectMissingSecret(&d, plan, cfg, typ)
	return d
}

func rejectMissingSecret(d *diag.Diagnostics, plan, cfg s3StorageResourceModel, typ string) {
	if typ == "sftp" && (stringConfigured(cfg.PrivateKeyUUID) || stringConfigured(plan.PrivateKeyUUID)) {
		return
	}
	if stringConfigured(cfg.Secret) || stringConfigured(plan.Secret) {
		return
	}
	d.AddAttributeError(path.Root("secret"), "secret is required", "Set secret. For sftp, set private_key_uuid instead to log in with a team private key.")
}

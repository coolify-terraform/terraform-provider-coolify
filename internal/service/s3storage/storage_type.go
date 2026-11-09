package s3storage

import (
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

func boolConfigured(v types.Bool) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// rejectStoragePlan stops create and update before a request Coolify 4.7
// would reject. Non-s3 types cannot send bucket, region, path-style, or
// skip-tls. s3 still requires bucket and region.
func rejectStoragePlan(c *client.Client, plan s3StorageResourceModel) diag.Diagnostics {
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
		if stringConfigured(plan.PrivateKeyUUID) {
			d.AddAttributeError(path.Root("private_key_uuid"), "private_key_uuid is only for sftp", "Remove private_key_uuid, or set type to sftp.")
		}
		return d
	}
	if stringConfigured(plan.Bucket) {
		d.AddAttributeError(path.Root("bucket"), "bucket is only for s3 storage", fmt.Sprintf("Remove bucket when type is %s. Coolify rejects the field.", typ))
	}
	if stringConfigured(plan.Region) {
		d.AddAttributeError(path.Root("region"), "region is only for s3 storage", fmt.Sprintf("Remove region when type is %s. Coolify rejects the field.", typ))
	}
	if boolConfigured(plan.UsePathStyleEndpoint) {
		d.AddAttributeError(path.Root("use_path_style_endpoint"), "use_path_style_endpoint is only for s3 storage", fmt.Sprintf("Remove use_path_style_endpoint when type is %s.", typ))
	}
	if boolConfigured(plan.SkipTLSVerification) {
		d.AddAttributeError(path.Root("skip_tls_verification"), "skip_tls_verification is only for s3 storage", fmt.Sprintf("Remove skip_tls_verification when type is %s.", typ))
	}
	if typ != "sftp" && stringConfigured(plan.PrivateKeyUUID) {
		d.AddAttributeError(path.Root("private_key_uuid"), "private_key_uuid is only for sftp", fmt.Sprintf("Remove private_key_uuid when type is %s.", typ))
	}
	return d
}

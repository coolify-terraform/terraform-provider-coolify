package application

import (
	"fmt"
	"sort"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// versionGatedWriteAttrSummary is the shared human-facing summary for #663.
const versionGatedWriteAttrSummary = "Coolify version cannot write some application settings"

// configuredVersionGatedWriteAttrs returns Terraform attribute names that the
// practitioner has set (non-null, non-unknown) and that Coolify only accepts on
// write when SupportsApplicationSettings is true (Coolify >= v4.2.0).
//
// Must stay aligned with client.UpdateApplicationInput.clearApplicationSettings.
func configuredVersionGatedWriteAttrs(f commonAppFields) []string {
	var names []string
	appendBool := func(name string, v *types.Bool) {
		if v != nil && !v.IsNull() && !v.IsUnknown() {
			names = append(names, name)
		}
	}
	appendInt := func(name string, v *types.Int64) {
		if v != nil && !v.IsNull() && !v.IsUnknown() {
			names = append(names, name)
		}
	}

	// v4.2.0 literal allow-list fields.
	appendBool("is_preview_deployments_enabled", f.IsPreviewDeploymentsEnabled)
	appendBool("use_build_secrets", f.UseBuildSecrets)
	// APPLICATION_SETTING_FIELDS.
	appendBool("is_git_submodules_enabled", f.IsGitSubmodulesEnabled)
	appendBool("is_git_lfs_enabled", f.IsGitLfsEnabled)
	appendBool("is_git_shallow_clone_enabled", f.IsGitShallowCloneEnabled)
	appendBool("disable_build_cache", f.DisableBuildCache)
	appendBool("inject_build_args_to_dockerfile", f.InjectBuildArgsToDockerfile)
	appendBool("include_source_commit_in_build", f.IncludeSourceCommitInBuild)
	appendBool("is_env_sorting_enabled", f.IsEnvSortingEnabled)
	appendBool("is_pr_deployments_public_enabled", f.IsPrDeploymentsPublicEnabled)
	appendInt("stop_grace_period", f.StopGracePeriod)
	appendInt("docker_images_to_keep", f.DockerImagesToKeep)
	appendBool("is_gzip_enabled", f.IsGzipEnabled)
	appendBool("is_stripprefix_enabled", f.IsStripprefixEnabled)
	appendBool("is_raw_compose_deployment_enabled", f.IsRawComposeDeploymentEnabled)

	sort.Strings(names)
	return names
}

// warnUnsupportedApplicationSettingsWrites adds a plan/apply warning when the
// connected Coolify is older than v4.2.0 and config sets fields that the API
// will not accept. Values stay in Terraform state; the client strips them on
// PATCH (see #662 / #663).
func warnUnsupportedApplicationSettingsWrites(c *client.Client, f commonAppFields, diags *diag.Diagnostics) {
	if diags == nil {
		return
	}
	if c == nil || c.SupportsApplicationSettings() {
		return
	}
	names := configuredVersionGatedWriteAttrs(f)
	if len(names) == 0 {
		return
	}
	ver := c.CoolifyVersion
	if ver == "" {
		ver = "unknown"
	}
	diags.AddWarning(
		versionGatedWriteAttrSummary,
		fmt.Sprintf(
			"This Coolify instance (%s) is older than v4.2.0, which is required to write: %s. "+
				"The provider will keep these values in Terraform state but will not send them to the Coolify API. "+
				"Upgrade Coolify to v4.2.0 or later, or remove these attributes from configuration.",
			ver,
			strings.Join(names, ", "),
		),
	)
}

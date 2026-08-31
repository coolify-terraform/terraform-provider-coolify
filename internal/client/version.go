package client

import (
	"strconv"
	"strings"
)

// minApplicationSettingsVersion is the first Coolify release whose application
// endpoints accept ApplicationSetting fields.
//
// Coolify builds the allow list for both create_application and
// update_by_uuid as a literal list ending in
// `...self::APPLICATION_SETTING_FIELDS`. That constant does not exist before
// v4.2.0, so on v4.1.x every settings field falls outside the allow list and
// comes back 422 "This field is not allowed." — and one rejected field aborts
// the entire request, which fails Create and leaves the resource tainted.
const minApplicationSettingsVersion = "4.2.0"

// ApplicationSettingsWriteJSONKeys lists Coolify application PATCH JSON keys
// accepted only on >= v4.2.0 (the v4.2.0 APPLICATION_SETTING_FIELDS set plus
// the two v4.2.0 literals is_preview_deployments_enabled and use_build_secrets).
//
// clearApplicationSettings and the plan-time warning list in the application
// service package must stay aligned with this slice. Prefer this list in tests
// over re-declaring the keys.
//
// Coolify v4.3.0 added more APPLICATION_SETTING_FIELDS and noindex_domains;
// those live in ApplicationSettingsV43WriteJSONKeys.
var ApplicationSettingsWriteJSONKeys = []string{
	"is_preview_deployments_enabled",
	"use_build_secrets",
	"is_git_submodules_enabled",
	"is_git_lfs_enabled",
	"is_git_shallow_clone_enabled",
	"disable_build_cache",
	"inject_build_args_to_dockerfile",
	"include_source_commit_in_build",
	"is_env_sorting_enabled",
	"is_pr_deployments_public_enabled",
	"stop_grace_period",
	"docker_images_to_keep",
	"is_gzip_enabled",
	"is_stripprefix_enabled",
	"is_raw_compose_deployment_enabled",
}

// minApplicationSettingsV43Version is the first Coolify release whose
// application endpoints accept the v4.3.0 APPLICATION_SETTING_FIELDS additions
// (log drain, GPU, consistent container name, custom_internal_name) and
// noindex_domains on the Application model.
const minApplicationSettingsV43Version = "4.3.0"

// ApplicationSettingsV43WriteJSONKeys lists Coolify application PATCH JSON keys
// accepted only on >= v4.3.0. clearApplicationSettingsV43 and the plan-time
// warning list must stay aligned with this slice.
var ApplicationSettingsV43WriteJSONKeys = []string{
	"is_log_drain_enabled",
	"is_gpu_enabled",
	"gpu_driver",
	"gpu_count",
	"gpu_device_ids",
	"gpu_options",
	"is_consistent_container_name_enabled",
	"custom_internal_name",
	"noindex_domains",
	"max_restart_count",
}

// SupportsApplicationSettings reports whether the connected instance accepts
// Coolify >= v4.2.0 application write fields (APPLICATION_SETTING_FIELDS plus
// is_preview_deployments_enabled and use_build_secrets, which landed as
// literals on the same endpoints in v4.2.0).
//
// An unknown version (empty CoolifyVersion) reports true: the provider already
// refuses to configure against an instance below minCoolifyVersion, so the only
// way to reach here without a version is a path that never called Configure
// (unit tests, mostly). Assuming the newest behaviour there keeps those tests
// exercising the full payload.
func (c *Client) SupportsApplicationSettings() bool {
	if c == nil || c.CoolifyVersion == "" {
		return true
	}
	return IsVersionAtLeast(c.CoolifyVersion, minApplicationSettingsVersion)
}

// SupportsApplicationSettingsV43 reports whether the connected instance accepts
// Coolify >= v4.3.0 application write fields (log drain, GPU settings,
// consistent container name, custom_internal_name, noindex_domains, max_restart_count).
//
// Empty CoolifyVersion reports true (same rationale as SupportsApplicationSettings).
func (c *Client) SupportsApplicationSettingsV43() bool {
	if c == nil || c.CoolifyVersion == "" {
		return true
	}
	return IsVersionAtLeast(c.CoolifyVersion, minApplicationSettingsV43Version)
}

// minSMTPEhloDomainVersion is the first Coolify version string that accepts
// smtp_ehlo_domain on GET/PATCH /notifications/email.
//
// The field landed on branch v4.x in coollabsio/coolify#11398 after tag
// v4.3.9 and is in git tag v4.3.10 (and 4.4-rc.1). Sending the key to
// v4.3.9 extra-key 422s.
const minSMTPEhloDomainVersion = "4.3.10"

// versionStringLagsTip reports GET /api/v1/version values that Coolify
// :edge has used while already shipping later tip APIs.
//
// CI edge has reported "4.3.0" after tag v4.3.10 fields such as
// smtp_ehlo_domain landed. Treat that string as "maybe tip" so the
// provider sends the field. A real 4.3.0 install extra-key 422s if the
// user sets a tip-only attribute.
func versionStringLagsTip(ver string) bool {
	v := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ver)), "v")
	return strings.HasPrefix(v, "4.3.0")
}

// SupportsSMTPEhloDomain reports whether the connected instance accepts
// smtp_ehlo_domain on email notification GET/PATCH.
//
// Empty CoolifyVersion reports true (same rationale as SupportsApplicationSettings).
// A 4.3.0 version string also reports true: CI edge lies with that
// string while shipping later tip fields. Acc tests must still extra-key
// probe before writing so a real 4.3.0 extra-key 422 is skipped.
func (c *Client) SupportsSMTPEhloDomain() bool {
	if c == nil || c.CoolifyVersion == "" {
		return true
	}
	if versionStringLagsTip(c.CoolifyVersion) {
		return true
	}
	return IsVersionAtLeast(c.CoolifyVersion, minSMTPEhloDomainVersion)
}

// IsVersionAtLeast compares two semver-like version strings (e.g. "4.0.0").
// Returns true if actual >= minimum. Non-parseable versions return true
// to avoid blocking on unexpected version formats.
func IsVersionAtLeast(actual, minimum string) bool {
	parse := func(v string) (int, int, int, bool) {
		v = strings.TrimPrefix(v, "v")
		parts := strings.SplitN(v, ".", 3)
		if len(parts) < 2 {
			return 0, 0, 0, false
		}
		major, err1 := strconv.Atoi(parts[0])
		minor, err2 := strconv.Atoi(parts[1])
		patch := 0
		if len(parts) == 3 {
			// Strip pre-release suffix (e.g. "0-beta.335")
			p := strings.SplitN(parts[2], "-", 2)[0]
			patch, _ = strconv.Atoi(p)
		}
		if err1 != nil || err2 != nil {
			return 0, 0, 0, false
		}
		return major, minor, patch, true
	}
	aMaj, aMin, aPat, aOk := parse(actual)
	mMaj, mMin, mPat, mOk := parse(minimum)
	if !aOk || !mOk {
		return true
	}
	if aMaj != mMaj {
		return aMaj > mMaj
	}
	if aMin != mMin {
		return aMin > mMin
	}
	return aPat >= mPat
}

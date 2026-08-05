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

// SupportsApplicationSettings reports whether the connected instance accepts
// ApplicationSetting fields on the application write endpoints.
//
// An unknown version (empty CoolifyVersion) reports true: the provider already
// refuses to configure against an instance below minCoolifyVersion, so the only
// way to reach here without a version is a path that never called Configure —
// unit tests, mostly. Assuming the newest behaviour there keeps those tests
// exercising the full payload.
func (c *Client) SupportsApplicationSettings() bool {
	if c == nil || c.CoolifyVersion == "" {
		return true
	}
	return IsVersionAtLeast(c.CoolifyVersion, minApplicationSettingsVersion)
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

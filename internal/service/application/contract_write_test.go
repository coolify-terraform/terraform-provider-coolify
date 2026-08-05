package application

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The contract tests in internal/spectest check one direction only: that every
// field Coolify declares is covered by the client structs. Nothing checked the
// reverse — that the provider never *sends* a field Coolify refuses.
//
// PATCH /applications/{uuid} validates with
// `array_diff(array_keys($request->all()), $allowedFields)` and returns 422
// "This field is not allowed." for every extra key. One rejected key fails the
// whole request, so a single stray field breaks Create for the whole resource.
//
// The tests below close that direction for the two request builders. They check
// against the allow list of the NEWEST supported Coolify: withholding fields on
// older instances is the version gate's job, covered by
// TestUpdateApplication_VersionGate in internal/client.

// applicationSettingFields is Coolify's APPLICATION_SETTING_FIELDS constant.
//
// It is spelled out here rather than read from the contract because the
// extractor collects only string literals from `$allowedFields = [...]` and
// does not expand the trailing `...self::APPLICATION_SETTING_FIELDS` spread, so
// the generated contracts understate the allow list by exactly these entries.
// Tracked as #661; once the extractor expands spreads, drop this list and let
// updateAllowedFields carry them.
var applicationSettingFields = []string{
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

func readContractAllowList(t *testing.T) map[string]bool {
	t.Helper()

	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "testdata", "contracts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading contracts dir: %v", err)
	}
	var latest string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			latest = e.Name()
		}
	}
	if latest == "" {
		t.Fatal("no contract JSON found in testdata/contracts/")
	}
	data, err := os.ReadFile(filepath.Join(dir, latest))
	if err != nil {
		t.Fatalf("reading contract: %v", err)
	}
	var contract struct {
		Endpoints map[string]struct {
			AllowedFields []string `json:"allowed_fields"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("parsing contract: %v", err)
	}
	ep, ok := contract.Endpoints["ApplicationsController::update_by_uuid"]
	if !ok {
		t.Fatal("ApplicationsController::update_by_uuid not found in contract")
	}
	if len(ep.AllowedFields) == 0 {
		t.Fatal("update_by_uuid has an empty allow list")
	}
	allowed := make(map[string]bool, len(ep.AllowedFields))
	for _, f := range ep.AllowedFields {
		allowed[f] = true
	}
	return allowed
}

// updateAllowedFields returns the allow list Coolify >= 4.2.0 applies to
// PATCH /applications/{uuid}: the literals from the pinned contract, plus the
// settings spread the extractor misses (see #661).
func updateAllowedFields(t *testing.T) map[string]bool {
	t.Helper()

	allowed := readContractAllowList(t)
	for _, f := range applicationSettingFields {
		allowed[f] = true
	}
	return allowed
}

// fillCommonAppFields populates every pointer field of commonAppFields, so the
// builders emit as many keys as they can. Using reflection rather than a
// literal keeps this test honest as fields are added: a new attribute wired
// into a builder is covered here the day it lands.
func fillCommonAppFields(t *testing.T, str string, num int64, b bool) commonAppFields {
	t.Helper()

	var f commonAppFields
	v := reflect.ValueOf(&f).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() != reflect.Pointer || !field.CanSet() {
			continue
		}
		switch field.Type() {
		case reflect.TypeOf((*types.String)(nil)):
			val := types.StringValue(str)
			field.Set(reflect.ValueOf(&val))
		case reflect.TypeOf((*types.Int64)(nil)):
			val := types.Int64Value(num)
			field.Set(reflect.ValueOf(&val))
		case reflect.TypeOf((*types.Bool)(nil)):
			val := types.BoolValue(b)
			field.Set(reflect.ValueOf(&val))
		}
	}
	// docker_compose_domains is carried to the wire as a JSON array, so the
	// generic probe string above would not survive marshalling (#658).
	domains := types.StringValue(`[{"name":"web","domain":"http://` + str + `.example.com"}]`)
	f.DockerComposeDomains = &domains
	return f
}

// disallowedKeys marshals an input and returns the keys Coolify would reject.
func disallowedKeys(t *testing.T, input any, allowed map[string]bool) []string {
	t.Helper()

	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshalling input: %v", err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshalling input: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("input marshalled to an empty body; the builder emitted nothing")
	}
	var bad []string
	for key := range body {
		if !allowed[key] {
			bad = append(bad, key)
		}
	}
	sort.Strings(bad)
	return bad
}

func TestPostCreatePatch_OnlySendsAllowedFields(t *testing.T) {
	t.Parallel()

	allowed := updateAllowedFields(t)
	f := fillCommonAppFields(t, "contract-probe", 7, true)

	if bad := disallowedKeys(t, buildPostCreatePatch(f), allowed); len(bad) > 0 {
		t.Errorf("post-create PATCH sends %d field(s) outside the update_by_uuid allow list; "+
			"Coolify answers 422 \"This field is not allowed.\" and aborts the whole request:\n  %s",
			len(bad), strings.Join(bad, "\n  "))
	}
}

func TestUpdateInput_OnlySendsAllowedFields(t *testing.T) {
	t.Parallel()

	allowed := updateAllowedFields(t)
	// Differing plan and state so every diff-guarded field is emitted.
	plan := fillCommonAppFields(t, "contract-probe-plan", 7, true)
	state := fillCommonAppFields(t, "contract-probe-state", 9, false)

	bad := disallowedKeys(t, buildUpdateInput(plan, state), allowed)

	// Known gaps, tracked separately: these are not ApplicationSetting fields
	// and removing them would drop practitioner-visible behaviour, so they need
	// their own decision rather than being silently dropped here.
	knownGaps := map[string]string{
		"dockerfile":           "create-only field; editing a Dockerfile in place has no allowed update route",
		"docker_compose_raw":   "create-only field, and only for the deprecated inline-compose endpoint",
		"github_app_uuid":      "create-only field; re-pointing an app at another GitHub App has no update route",
		"preview_url_template": "absent from both allow lists; no write route at all",
	}
	var unexpected []string
	for _, key := range bad {
		if _, known := knownGaps[key]; !known {
			unexpected = append(unexpected, key)
		}
	}
	if len(unexpected) > 0 {
		t.Errorf("update PATCH sends %d unexpected field(s) outside the update_by_uuid allow list:\n  %s",
			len(unexpected), strings.Join(unexpected, "\n  "))
	}

	// Guard the gap list itself: once a field gains a write route, drop it here.
	for key, why := range knownGaps {
		if allowed[key] {
			t.Errorf("%q is on the allow list now (%s); remove it from knownGaps", key, why)
		}
	}
}

// TestApplicationSettingFields_AbsentFromExtractedContract pins the #661 bug in
// place. The moment the extractor learns to expand PHP spreads, this fails, and
// whoever fixes it can delete applicationSettingFields above and read the whole
// allow list from the contract instead.
func TestApplicationSettingFields_AbsentFromExtractedContract(t *testing.T) {
	t.Parallel()

	literals := readContractAllowList(t)
	var present []string
	for _, f := range applicationSettingFields {
		if literals[f] {
			present = append(present, f)
		}
	}
	if len(present) > 0 {
		t.Errorf("the extractor now resolves the APPLICATION_SETTING_FIELDS spread (#661 fixed for %d field(s): %s).\n"+
			"Delete applicationSettingFields in this file and read the allow list from the contract alone.",
			len(present), strings.Join(present, ", "))
	}
}

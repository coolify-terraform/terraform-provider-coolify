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
// against the allow list of the NEWEST supported Coolify (pin / v4.2.0), which
// includes expanded ...self::APPLICATION_SETTING_FIELDS spreads (#661).
// Withholding settings on older instances is the version gate's job
// (TestUpdateApplication_VersionGate).

func readContractAllowList(t *testing.T) map[string]bool {
	t.Helper()

	// Prefer the pin (coolify-v4.json); fall back to newest versioned file.
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "testdata", "contracts")
	candidates := []string{
		filepath.Join(dir, "coolify-v4.json"),
		filepath.Join(dir, "coolify-v4.2.0.json"),
	}
	var data []byte
	var err error
	for _, path := range candidates {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
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

// TestContractAllowList_IncludesApplicationSettingFields locks #661: the
// extractor expands ...self::APPLICATION_SETTING_FIELDS so the pin lists them.
func TestContractAllowList_IncludesApplicationSettingFields(t *testing.T) {
	t.Parallel()

	allowed := readContractAllowList(t)
	required := []string{
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
		// Coolify >= v4.3.0 APPLICATION_SETTING_FIELDS + noindex_domains
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
	var missing []string
	for _, f := range required {
		if !allowed[f] {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		t.Errorf("contract update_by_uuid missing APPLICATION_SETTING_FIELDS after #661: %s",
			strings.Join(missing, ", "))
	}
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

	allowed := readContractAllowList(t)
	f := fillCommonAppFields(t, "contract-probe", 7, true)

	if bad := disallowedKeys(t, buildPostCreatePatch(f), allowed); len(bad) > 0 {
		t.Errorf("post-create PATCH sends %d field(s) outside the update_by_uuid allow list; "+
			"Coolify answers 422 \"This field is not allowed.\" and aborts the whole request:\n  %s",
			len(bad), strings.Join(bad, "\n  "))
	}
}

func TestUpdateInput_OnlySendsAllowedFields(t *testing.T) {
	t.Parallel()

	allowed := readContractAllowList(t)
	// Differing plan and state so every diff-guarded field is emitted.
	plan := fillCommonAppFields(t, "contract-probe-plan", 7, true)
	state := fillCommonAppFields(t, "contract-probe-state", 9, false)

	bad := disallowedKeys(t, buildUpdateInput(plan, state), allowed)

	// Known gaps, tracked separately: these are not ApplicationSetting fields
	// and removing them would drop practitioner-visible behaviour, so they need
	// their own decision rather than being silently dropped here.
	// preview_url_template is on the v4.3.0 update allow list (and already sent
	// when changed via shared update builders).
	knownGaps := map[string]string{
		"dockerfile":         "create-only field; editing a Dockerfile in place has no allowed update route",
		"docker_compose_raw": "create-only field, and only for the deprecated inline-compose endpoint",
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

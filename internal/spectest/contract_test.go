package spectest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
)

// contractFile is the contract JSON extracted from Coolify source code.
// This is the single source of truth for field definitions, defaults,
// types, and sensitive markings.
type contractFile struct {
	Version            string                   `json:"version"`
	Models             map[string]contractModel `json:"models"`
	Enums              map[string][]string      `json:"enums"`
	ValidationPatterns map[string]string        `json:"validation_patterns"`
}

type contractModel struct {
	Table          string                   `json:"table"`
	Fillable       []string                 `json:"fillable"`
	Hidden         []string                 `json:"hidden"`
	Fields         map[string]contractField `json:"fields"`
	SettingsFields map[string]contractField `json:"settings_fields"`
}

type contractField struct {
	Type       string      `json:"type"`
	Nullable   bool        `json:"nullable"`
	Default    interface{} `json:"default"`
	Cast       *string     `json:"cast"`
	Sensitive  bool        `json:"sensitive"`
	Fillable   bool        `json:"fillable"`
	MaxLength  *int        `json:"max_length,omitempty"`
	EnumValues []string    `json:"enum_values,omitempty"`
}

func loadContract(t *testing.T) contractFile {
	t.Helper()
	dir := filepath.Join(testdataDir(), "contracts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading contracts dir: %v", err)
	}
	// Find the latest contract file
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
	var c contractFile
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("parsing contract: %v", err)
	}
	return c
}

// jsonTagsFromStruct extracts all json tags from a struct type (non-recursive).
func jsonTagsFromStruct(t reflect.Type) map[string]reflect.StructField {
	tags := make(map[string]reflect.StructField)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name != "" {
			tags[name] = f
		}
	}
	return tags
}

// fieldsToIgnore are contract fields that are intentionally not in our client
// structs because they are internal-only (DB IDs, morphs, computed, etc.).
var fieldsToIgnore = map[string]map[string]bool{
	"Application": {
		// Internal DB identifiers (not exposed via API)
		"environment_id":        true,
		"destination_id":        true,
		"destination_type":      true,
		"source_id":             true,
		"source_type":           true,
		"private_key_id":        true,
		"repository_project_id": true,
		// Computed/internal fields not exposed by the API
		"config_hash":              true,
		"custom_healthcheck_found": true,
		"compose_parsing_version":  true,
		"last_online_at":           true,
		"last_restart_at":          true,
		"last_restart_type":        true,
		"restart_count":            true,
		"nixpkgsarchive":           true,
		"git_full_url":             true,
		// Docker compose PR fields (removed in later migrations)
		"docker_compose_pr_location": true,
		"docker_compose_pr":          true,
		"docker_compose_pr_raw":      true,
		// Fields served from related models, not Application table
		"docker_compose": true,
		// Swarm-only fields (not commonly used)
		"swarm_replicas":              true,
		"swarm_placement_constraints": true,
		// Internal-only ApplicationSetting fields not exposed by update API
		"application_id":                       true,
		"custom_internal_name":                 true,
		"disable_build_cache":                  true,
		"docker_images_to_keep":                true,
		"gpu_count":                            true,
		"gpu_device_ids":                       true,
		"gpu_driver":                           true,
		"gpu_options":                          true,
		"include_source_commit_in_build":       true,
		"inject_build_args_to_dockerfile":      true,
		"is_consistent_container_name_enabled": true,
		"is_container_label_readonly_enabled":  true,
		"is_custom_ssl":                        true,
		"is_debug_enabled":                     true,
		"is_dual_cert":                         true,
		"is_env_sorting_enabled":               true,
		"is_git_lfs_enabled":                   true,
		"is_git_shallow_clone_enabled":         true,
		"is_git_submodules_enabled":            true,
		"is_gpu_enabled":                       true,
		"is_gzip_enabled":                      true,
		"is_http2":                             true,
		"is_include_timestamps":                true,
		"is_log_drain_enabled":                 true,
		"is_pr_deployments_public_enabled":     true,
		"is_preview_deployments_enabled":       true,
		"is_raw_compose_deployment_enabled":    true,
		"is_stripprefix_enabled":               true,
		"is_swarm_only_worker_nodes":           true,
		"use_build_secrets":                    true,
		// is_build_server_enabled is the setting name; the API field is use_build_server (already in struct)
		"is_build_server_enabled": true,
	},
	"Server": {
		"proxy":                         true,
		"traefik_outdated_info":         true,
		"server_metadata":               true,
		"logdrain_axiom_api_key":        true,
		"logdrain_newrelic_license_key": true,
		"delete_unused_volumes":         true,
		"delete_unused_networks":        true,
		"unreachable_notification_sent": true,
		"unreachable_count":             true,
		"validation_logs":               true,
		"hetzner_server_id":             true,
		"hetzner_server_status":         true,
		"is_validating":                 true,
		"detected_traefik_version":      true,
		"ip_previous":                   true,
		"sentinel_updated_at":           true,
	},
}

// TestContractCoverage_Application checks that client.Application has JSON
// tags for all user-facing fillable fields in the contract.
func TestContractCoverage_Application(t *testing.T) {
	t.Parallel()
	c := loadContract(t)

	appContract, ok := c.Models["Application"]
	if !ok {
		t.Fatal("Application model not found in contract")
	}

	goTags := jsonTagsFromStruct(reflect.TypeOf(client.Application{}))
	ignore := fieldsToIgnore["Application"]

	var missing []string
	for fieldName, field := range appContract.Fields {
		if !field.Fillable {
			continue
		}
		if ignore[fieldName] {
			continue
		}
		if _, ok := goTags[fieldName]; !ok {
			missing = append(missing, fieldName)
		}
	}

	// Also check settings fields
	for fieldName := range appContract.SettingsFields {
		if ignore[fieldName] {
			continue
		}
		if _, ok := goTags[fieldName]; !ok {
			missing = append(missing, fieldName+" (setting)")
		}
	}

	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("client.Application is missing %d contract fields:\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// TestContractCoverage_Databases checks all database model structs.
func TestContractCoverage_Databases(t *testing.T) {
	t.Parallel()
	c := loadContract(t)
	goTags := jsonTagsFromStruct(reflect.TypeOf(client.Database{}))

	dbModels := []string{
		"StandalonePostgresql", "StandaloneMysql", "StandaloneMariadb",
		"StandaloneMongodb", "StandaloneRedis", "StandaloneClickhouse",
		"StandaloneKeydb", "StandaloneDragonfly",
	}

	dbIgnore := map[string]bool{
		"environment_id":    true,
		"destination_id":    true,
		"destination_type":  true,
		"started_at":        true,
		"last_online_at":    true,
		"last_restart_at":   true,
		"last_restart_type": true,
		"restart_count":     true,
	}

	for _, modelName := range dbModels {
		model, ok := c.Models[modelName]
		if !ok {
			t.Errorf("model %s not found in contract", modelName)
			continue
		}

		var missing []string
		for fieldName, field := range model.Fields {
			if !field.Fillable {
				continue
			}
			if dbIgnore[fieldName] {
				continue
			}
			if _, ok := goTags[fieldName]; !ok {
				missing = append(missing, fieldName)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("client.Database is missing %d fields from %s:\n  %s",
				len(missing), modelName, strings.Join(missing, "\n  "))
		}
	}
}

// TestContractCoverage_Sensitive checks that all encrypted fields in the
// contract are tracked. This does not check the Terraform schema (that
// requires schema introspection), but ensures awareness.
func TestContractCoverage_Sensitive(t *testing.T) {
	t.Parallel()
	c := loadContract(t)

	var sensitiveFields []string
	for modelName, model := range c.Models {
		for fieldName, field := range model.Fields {
			if field.Sensitive {
				sensitiveFields = append(sensitiveFields, modelName+"."+fieldName)
			}
		}
	}
	sort.Strings(sensitiveFields)
	t.Logf("Sensitive (encrypted) fields in contract (%d):\n  %s",
		len(sensitiveFields), strings.Join(sensitiveFields, "\n  "))
}

// contractCoverageTest is a reusable helper for model contract tests.
func contractCoverageTest(t *testing.T, modelName string, goType reflect.Type, ignore map[string]bool) {
	t.Helper()
	c := loadContract(t)
	model, ok := c.Models[modelName]
	if !ok {
		t.Fatalf("model %s not found in contract", modelName)
	}
	goTags := jsonTagsFromStruct(goType)
	if ignore == nil {
		ignore = map[string]bool{}
	}
	var missing []string
	for fieldName, field := range model.Fields {
		if !field.Fillable || ignore[fieldName] {
			continue
		}
		if _, ok := goTags[fieldName]; !ok {
			missing = append(missing, fieldName)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("client struct is missing %d contract fields from %s:\n  %s",
			len(missing), modelName, strings.Join(missing, "\n  "))
	}
}

func TestContractCoverage_Server(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "Server", reflect.TypeOf(client.Server{}), map[string]bool{
		"team_id":                   true,
		"private_key_id":            true,
		"proxy":                     true, // complex JSON object
		"sentinel_token":            true, // hidden by middleware
		"sentinel_custom_url":       true,
		"sentinel_metrics_token":    true,
		"sentinel_metrics_history":  true,
		"sentinel_metrics_interval": true,
		"started_at":                true,
		"last_online_at":            true,
		"last_restart_at":           true,
		"last_restart_type":         true,
		"restart_count":             true,
		"unreachable_notification":  true,
		"unreachable_count":         true,
		"log_drain_notification":    true,
		"swarm_cluster":             true,
		"cloud_provider_token_id":   true, // internal FK
		"detected_traefik_version":  true, // ephemeral status
		"hetzner_server_id":         true, // Hetzner-specific
		"hetzner_server_status":     true, // Hetzner-specific
		"ip_previous":               true, // internal tracking
		"is_validating":             true, // ephemeral status
		"server_metadata":           true, // internal metadata
		"traefik_outdated_info":     true, // ephemeral status
	})
}

func TestContractCoverage_Service(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "Service", reflect.TypeOf(client.Service{}), map[string]bool{
		"team_id":                             true,
		"environment_id":                      true,
		"destination_id":                      true,
		"destination_type":                    true,
		"server_id":                           true,
		"config_hash":                         true,
		"docker_compose_raw":                  true, // sensitive, hidden
		"docker_compose":                      true, // sensitive, hidden
		"connect_to_docker_network":           true,
		"is_container_label_escape_enabled":   true,
		"is_container_label_readonly_enabled": true,
		"is_readonly":                         true,
		"compose_parsing_version":             true, // internal config
		"service_type":                        true, // mapped to "type" in client struct
	})
}

func TestContractCoverage_PrivateKey(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "PrivateKey", reflect.TypeOf(client.PrivateKey{}), map[string]bool{
		"team_id": true,
	})
}

func TestContractCoverage_EnvironmentVariable(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "EnvironmentVariable", reflect.TypeOf(client.EnvironmentVariable{}), map[string]bool{
		"resourceable_id":   true,
		"resourceable_type": true,
		"team_id":           true,
		"real_value":        true, // computed accessor
		"version":           true,
		"comment":           true, // not exposed in provider
		"is_literal":        true, // internal flag
		"is_multiline":      true, // internal flag
		"is_required":       true, // internal flag
		"is_runtime":        true, // internal flag
		"is_shared":         true, // internal flag
		"is_shown_once":     true, // internal flag
		"order":             true, // UI ordering
	})
}

func TestContractCoverage_ScheduledTask(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "ScheduledTask", reflect.TypeOf(client.ScheduledTask{}), map[string]bool{
		"application_id": true,
		"service_id":     true,
		"team_id":        true,
		"container":      true, // not user-facing
		"timeout":        true, // not exposed yet
	})
}

func TestContractCoverage_Project(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "Project", reflect.TypeOf(client.Project{}), map[string]bool{
		"team_id": true,
	})
}

func TestContractCoverage_GithubApp(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "GithubApp", reflect.TypeOf(client.GitHubApp{}), map[string]bool{
		"team_id":        true,
		"private_key_id": true,
		"client_secret":  true, // sensitive, hidden
		"is_system_wide": true,
		"administration": true, // GitHub permission scope
		"contents":       true, // GitHub permission scope
		"custom_port":    true, // internal config
		"custom_user":    true, // internal config
		"is_public":      true, // internal flag
		"metadata":       true, // GitHub permission scope
		"pull_requests":  true, // GitHub permission scope
	})
}

func TestContractCoverage_ScheduledDatabaseBackup(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "ScheduledDatabaseBackup", reflect.TypeOf(client.DatabaseBackup{}), map[string]bool{
		"team_id":              true,
		"database_id":          true,
		"description":          true, // not exposed yet
		"disable_local_backup": true, // not exposed yet
		"s3_storage_id":        true, // numeric FK; provider uses s3_storage_uuid
	})
}

func TestContractCoverage_CloudToken(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "CloudProviderToken", reflect.TypeOf(client.CloudToken{}), map[string]bool{
		"team_id": true, // internal FK
	})
}

func TestContractCoverage_Storage(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "LocalPersistentVolume", reflect.TypeOf(client.Storage{}), map[string]bool{
		"container_id":              true, // internal Docker container ID
		"resource_id":               true, // numeric FK; provider uses resource_uuid
		"is_preview_suffix_enabled": true, // not exposed yet
	})
}

// TestContractCoverage_Report prints a summary of coverage. Run with -v.
func TestContractCoverage_Report(t *testing.T) {
	t.Parallel()
	c := loadContract(t)

	t.Logf("\n=== Contract Coverage Report (version: %s) ===", c.Version)
	t.Logf("Models: %d", len(c.Models))
	t.Logf("Enums: %d", len(c.Enums))
	t.Logf("Validation patterns: %d", len(c.ValidationPatterns))

	// Check Application coverage
	if app, ok := c.Models["Application"]; ok {
		goTags := jsonTagsFromStruct(reflect.TypeOf(client.Application{}))
		ignore := fieldsToIgnore["Application"]
		total := 0
		covered := 0
		for name, field := range app.Fields {
			if !field.Fillable || ignore[name] {
				continue
			}
			total++
			if _, ok := goTags[name]; ok {
				covered++
			}
		}
		pct := float64(covered) / float64(total) * 100
		t.Logf("Application: %d/%d fields covered (%.1f%%)", covered, total, pct)
	}
}

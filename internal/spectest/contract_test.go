package spectest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
)

// contractFile is the contract JSON extracted from Coolify source code.
// This is the single source of truth for field definitions, defaults,
// types, and sensitive markings.
type contractFile struct {
	Version            string                      `json:"version"`
	Models             map[string]contractModel    `json:"models"`
	Endpoints          map[string]contractEndpoint `json:"endpoints"`
	Enums              map[string][]string         `json:"enums"`
	ValidationPatterns map[string]string           `json:"validation_patterns"`
}

// contractEndpoint is a Coolify controller action with an allow list.
type contractEndpoint struct {
	AllowedFields []string `json:"allowed_fields"`
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

// Application / Server model skips live in contract_skips.go (taxonomy #622).

// TestContractCoverage_Application checks that client.Application has JSON
// tags for all user-facing fillable fields in the contract, and that the
// Go types are compatible with the contract types.
func TestContractCoverage_Application(t *testing.T) {
	t.Parallel()
	c := loadContract(t)

	appContract, ok := c.Models["Application"]
	if !ok {
		t.Fatal("Application model not found in contract")
	}

	goTags := jsonTagsFromStruct(reflect.TypeOf(client.Application{}))
	ignore := applicationFieldSkips

	var missing []string
	var typeMismatches []string
	for fieldName, field := range appContract.Fields {
		if !field.Fillable || isSkipped(ignore, fieldName) {
			continue
		}
		goField, ok := goTags[fieldName]
		if !ok {
			missing = append(missing, fieldName)
			continue
		}
		if err := checkTypeCompatibility(field, goField.Type); err != nil {
			typeMismatches = append(typeMismatches, fmt.Sprintf("%s: %s", fieldName, err))
		}
	}

	// Also check settings fields
	for fieldName := range appContract.SettingsFields {
		if isSkipped(ignore, fieldName) {
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
	sort.Strings(typeMismatches)
	if len(typeMismatches) > 0 {
		t.Errorf("client.Application has %d type mismatches:\n  %s",
			len(typeMismatches), strings.Join(typeMismatches, "\n  "))
	}
}

// TestContractCoverage_Databases checks all database model structs for
// both field name coverage and Go type compatibility.
func TestContractCoverage_Databases(t *testing.T) {
	t.Parallel()
	c := loadContract(t)
	goTags := jsonTagsFromStruct(reflect.TypeOf(client.Database{}))

	dbModels := []string{
		"StandalonePostgresql", "StandaloneMysql", "StandaloneMariadb",
		"StandaloneMongodb", "StandaloneRedis", "StandaloneClickhouse",
		"StandaloneKeydb", "StandaloneDragonfly",
	}

	dbIgnore := databaseModelSkips

	for _, modelName := range dbModels {
		model, ok := c.Models[modelName]
		if !ok {
			t.Errorf("model %s not found in contract", modelName)
			continue
		}

		var missing []string
		var typeMismatches []string
		for fieldName, field := range model.Fields {
			if !field.Fillable || isSkipped(dbIgnore, fieldName) {
				continue
			}
			goField, ok := goTags[fieldName]
			if !ok {
				missing = append(missing, fieldName)
				continue
			}
			if err := checkTypeCompatibility(field, goField.Type); err != nil {
				typeMismatches = append(typeMismatches, fmt.Sprintf("%s: %s", fieldName, err))
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("client.Database is missing %d fields from %s:\n  %s",
				len(missing), modelName, strings.Join(missing, "\n  "))
		}
		sort.Strings(typeMismatches)
		if len(typeMismatches) > 0 {
			t.Errorf("client.Database has %d type mismatches in %s:\n  %s",
				len(typeMismatches), modelName, strings.Join(typeMismatches, "\n  "))
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

// checkTypeCompatibility validates that a Go struct field type is compatible
// with the contract's declared type and cast. Returns an error description
// on mismatch, nil when compatible.
func checkTypeCompatibility(field contractField, goType reflect.Type) error {
	// Dereference pointers to get the base type.
	base := goType
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}

	cast := ""
	if field.Cast != nil {
		cast = *field.Cast
	}

	switch {
	case field.Type == "json" && cast == "array":
		// json.RawMessage (which is []byte) is the correct Go type.
		// Reject plain string, which would fail json.Unmarshal on arrays.
		if base.Kind() == reflect.String {
			return fmt.Errorf("contract type json (cast:array) requires json.RawMessage, got string")
		}

	case field.Type == "boolean" || (field.Type == "boolean" && cast == "boolean"):
		if base.Kind() != reflect.Bool {
			return fmt.Errorf("contract type boolean requires bool, got %s", base.Kind())
		}

	case field.Type == "integer" || field.Type == "bigInteger":
		switch base.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			// numeric types are acceptable
		default:
			return fmt.Errorf("contract type %s requires numeric Go type, got %s", field.Type, base.Kind())
		}
	}
	// string, text, longText, enum, timestamp, and cast variants (integer,
	// float, encrypted, datetime, string) are all represented as Go strings.
	// No check needed for those.
	return nil
}

// contractCoverageTest is a reusable helper for model contract tests.
// It checks both field name coverage and Go type compatibility.
// ignore uses the skip taxonomy (#622): internal / deferred+#N / n/a.
func contractCoverageTest(t *testing.T, modelName string, goType reflect.Type, ignore map[string]FieldSkip) {
	t.Helper()
	c := loadContract(t)
	model, ok := c.Models[modelName]
	if !ok {
		t.Fatalf("model %s not found in contract", modelName)
	}
	goTags := jsonTagsFromStruct(goType)
	if ignore == nil {
		ignore = map[string]FieldSkip{}
	}
	requireValidSkips(t, ignore)

	var missing []string
	var typeMismatches []string

	for fieldName, field := range model.Fields {
		if !field.Fillable || isSkipped(ignore, fieldName) {
			continue
		}
		goField, ok := goTags[fieldName]
		if !ok {
			missing = append(missing, fieldName)
			continue
		}
		if err := checkTypeCompatibility(field, goField.Type); err != nil {
			typeMismatches = append(typeMismatches, fmt.Sprintf("%s: %s", fieldName, err))
		}
	}

	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("client struct is missing %d contract fields from %s:\n  %s",
			len(missing), modelName, strings.Join(missing, "\n  "))
	}
	sort.Strings(typeMismatches)
	if len(typeMismatches) > 0 {
		t.Errorf("client struct has %d type mismatches in %s:\n  %s",
			len(typeMismatches), modelName, strings.Join(typeMismatches, "\n  "))
	}
}

func TestContractCoverage_Server(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "Server", reflect.TypeOf(client.Server{}), serverCoverageSkips)
}

func TestContractCoverage_Service(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "Service", reflect.TypeOf(client.Service{}), serviceCoverageSkips)
}

func TestContractCoverage_PrivateKey(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "PrivateKey", reflect.TypeOf(client.PrivateKey{}), privateKeyCoverageSkips)
}

func TestContractCoverage_EnvironmentVariable(t *testing.T) {
	t.Parallel()
	// is_runtime, is_literal, is_multiline, comment, is_buildtime covered on client (#619)
	contractCoverageTest(t, "EnvironmentVariable", reflect.TypeOf(client.EnvironmentVariable{}), environmentVariableCoverageSkips)
}

func TestContractCoverage_ScheduledTask(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "ScheduledTask", reflect.TypeOf(client.ScheduledTask{}), scheduledTaskCoverageSkips)
}

func TestContractCoverage_Project(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "Project", reflect.TypeOf(client.Project{}), projectCoverageSkips)
}

func TestContractCoverage_GithubApp(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "GithubApp", reflect.TypeOf(client.GitHubApp{}), githubAppCoverageSkips)
}

func TestContractCoverage_ScheduledDatabaseBackup(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "ScheduledDatabaseBackup", reflect.TypeOf(client.DatabaseBackup{}), databaseBackupCoverageSkips)
}

func TestContractCoverage_CloudToken(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "CloudProviderToken", reflect.TypeOf(client.CloudToken{}), cloudTokenCoverageSkips)
}

func TestContractCoverage_Storage(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "LocalPersistentVolume", reflect.TypeOf(client.Storage{}), storageCoverageSkips)
}

func TestContractCoverage_ServerSetting(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "ServerSetting", reflect.TypeOf(client.ServerSettings{}), serverSettingCoverageSkips)
}

func TestContractCoverage_Environment(t *testing.T) {
	t.Parallel()
	contractCoverageTest(t, "Environment", reflect.TypeOf(client.Environment{}), environmentCoverageSkips)
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
		ignore := applicationFieldSkips
		total := 0
		covered := 0
		for name, field := range app.Fields {
			if !field.Fillable || isSkipped(ignore, name) {
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

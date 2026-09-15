package spectest

import (
	"sort"
	"strings"
	"testing"
)

func TestSchemaCoverage_EnvironmentVariable(t *testing.T) {
	t.Parallel()
	c := loadContract(t)
	model, ok := c.Models["EnvironmentVariable"]
	if !ok {
		t.Fatal("EnvironmentVariable model missing from contract")
	}

	byField := make(map[string]SchemaCoverageEntry, len(environmentVariableSchemaRegistry))
	for _, e := range environmentVariableSchemaRegistry {
		if err := e.validate(); err != nil {
			t.Errorf("registry entry %q: %v", e.ContractField, err)
			continue
		}
		if _, dup := byField[e.ContractField]; dup {
			t.Errorf("duplicate registry entry for %q", e.ContractField)
		}
		byField[e.ContractField] = e
	}

	// Every fillable contract field must have a registry row.
	var missingRegistry []string
	for name, field := range model.Fields {
		if !field.Fillable {
			continue
		}
		if _, ok := byField[name]; !ok {
			missingRegistry = append(missingRegistry, name)
		}
	}
	sort.Strings(missingRegistry)
	if len(missingRegistry) > 0 {
		t.Errorf("EnvironmentVariable fillable fields missing from schema registry:\n  %s",
			strings.Join(missingRegistry, "\n  "))
	}

	// Covered rows must exist on coolify_environment_variable schema.
	attrs, err := resourceSchemaAttributeNames(coolifyEnvironmentVariableResource())
	if err != nil {
		t.Fatalf("resource schema: %v", err)
	}
	for _, e := range environmentVariableSchemaRegistry {
		if e.Status != StatusCovered {
			continue
		}
		if _, ok := attrs[e.SchemaAttribute]; !ok {
			t.Errorf("covered field %s maps to schema attribute %q which is missing on coolify_environment_variable",
				e.ContractField, e.SchemaAttribute)
		}
	}
}

func TestSchemaCoverageEntry_DeferredRequiresIssue(t *testing.T) {
	t.Parallel()
	err := (SchemaCoverageEntry{
		ContractField: "x",
		Status:        SkipDeferred,
		Issue:         0,
		Notes:         "gap",
	}).validate()
	if err == nil {
		t.Fatal("expected deferred without issue to fail")
	}
}

func TestSchemaCoverage_ApplicationSettings(t *testing.T) {
	t.Parallel()
	c := loadContract(t)
	model, ok := c.Models["Application"]
	if !ok {
		t.Fatal("Application model missing")
	}
	byField := make(map[string]SchemaCoverageEntry, len(applicationSettingsSchemaRegistry))
	for _, e := range applicationSettingsSchemaRegistry {
		if err := e.validate(); err != nil {
			t.Errorf("registry entry %q: %v", e.ContractField, err)
			continue
		}
		if _, dup := byField[e.ContractField]; dup {
			t.Errorf("duplicate registry entry for %q", e.ContractField)
		}
		byField[e.ContractField] = e
	}
	var missing []string
	for name, field := range model.SettingsFields {
		if !field.Fillable {
			continue
		}
		if _, ok := byField[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("Application settings_fields missing from schema registry:\n  %s",
			strings.Join(missing, "\n  "))
	}
	attrs, err := resourceSchemaAttributeNames(coolifyApplicationResource())
	if err != nil {
		t.Fatalf("application schema: %v", err)
	}
	for _, e := range applicationSettingsSchemaRegistry {
		if e.Status != StatusCovered {
			continue
		}
		if _, ok := attrs[e.SchemaAttribute]; !ok {
			t.Errorf("covered settings field %s maps to %q missing on coolify_application",
				e.ContractField, e.SchemaAttribute)
		}
	}
}

func TestSchemaCoverage_ApplicationAllowList(t *testing.T) {
	t.Parallel()
	c := loadContract(t)
	createEP, ok := c.Endpoints["ApplicationsController::create_application"]
	if !ok {
		t.Fatal("ApplicationsController::create_application missing from contract")
	}
	updateEP, ok := c.Endpoints["ApplicationsController::update_by_uuid"]
	if !ok {
		t.Fatal("ApplicationsController::update_by_uuid missing from contract")
	}

	byField := make(map[string]SchemaCoverageEntry, len(applicationAllowListSchemaRegistry))
	for _, e := range applicationAllowListSchemaRegistry {
		if err := e.validate(); err != nil {
			t.Errorf("registry entry %q: %v", e.ContractField, err)
			continue
		}
		if _, dup := byField[e.ContractField]; dup {
			t.Errorf("duplicate registry entry for %q", e.ContractField)
		}
		byField[e.ContractField] = e
	}

	allow := map[string]struct{}{}
	for _, f := range createEP.AllowedFields {
		allow[f] = struct{}{}
	}
	for _, f := range updateEP.AllowedFields {
		allow[f] = struct{}{}
	}

	var missing []string
	for name := range allow {
		if _, ok := byField[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("ApplicationsController allow-list fields missing from Phase C schema registry:\n  %s",
			strings.Join(missing, "\n  "))
	}

	var extra []string
	for name := range byField {
		if _, ok := allow[name]; !ok {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Errorf("Phase C registry fields not on create_application or update_by_uuid allow-list:\n  %s",
			strings.Join(extra, "\n  "))
	}

	attrs, err := applicationResourceSchemaUnion()
	if err != nil {
		t.Fatalf("application schema union: %v", err)
	}
	for _, e := range applicationAllowListSchemaRegistry {
		if e.Status != StatusCovered {
			if e.Status == SkipInternal {
				t.Errorf("public allow-list field %s must not be marked internal", e.ContractField)
			}
			continue
		}
		if _, ok := attrs[e.SchemaAttribute]; !ok {
			t.Errorf("covered allow-list field %s maps to %q missing on application resource schemas",
				e.ContractField, e.SchemaAttribute)
		}
	}
}

func TestSchemaCoverage_ScheduledTask(t *testing.T) {
	t.Parallel()
	c := loadContract(t)
	model, ok := c.Models["ScheduledTask"]
	if !ok {
		t.Fatal("ScheduledTask model missing")
	}
	byField := make(map[string]SchemaCoverageEntry, len(scheduledTaskSchemaRegistry))
	for _, e := range scheduledTaskSchemaRegistry {
		if err := e.validate(); err != nil {
			t.Errorf("registry entry %q: %v", e.ContractField, err)
			continue
		}
		if _, dup := byField[e.ContractField]; dup {
			t.Errorf("duplicate registry entry for %q", e.ContractField)
		}
		byField[e.ContractField] = e
	}
	var missing []string
	for name, field := range model.Fields {
		if !field.Fillable {
			continue
		}
		if _, ok := byField[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("ScheduledTask fillable fields missing from schema registry:\n  %s",
			strings.Join(missing, "\n  "))
	}
	attrs, err := resourceSchemaAttributeNames(coolifyScheduledTaskResource())
	if err != nil {
		t.Fatalf("scheduled_task schema: %v", err)
	}
	for _, e := range scheduledTaskSchemaRegistry {
		if e.Status != StatusCovered {
			continue
		}
		if _, ok := attrs[e.SchemaAttribute]; !ok {
			t.Errorf("covered field %s maps to %q missing on coolify_scheduled_task",
				e.ContractField, e.SchemaAttribute)
		}
	}
}

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

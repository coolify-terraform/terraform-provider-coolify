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

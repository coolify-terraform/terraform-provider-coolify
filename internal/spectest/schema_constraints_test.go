package spectest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// schemaValidatorBound is the provider's declared max_length or OneOf list.
// The schema must not be looser than the pinned contract (#879).
type schemaValidatorBound struct {
	Model     string
	Field     string
	MaxLength int      // 0 means not checked
	Enum      []string // nil means not checked
}

var schemaValidatorBounds = []schemaValidatorBound{
	{Model: "Application", Field: "docker_compose_custom_start_command", MaxLength: 255},
	{Model: "Application", Field: "docker_compose_custom_build_command", MaxLength: 255},
	{Model: "StandalonePostgresql", Field: "ssl_mode", Enum: []string{"allow", "prefer", "require", "verify-ca", "verify-full"}},
	{Model: "StandaloneMysql", Field: "ssl_mode", Enum: []string{"PREFERRED", "REQUIRED", "VERIFY_CA", "VERIFY_IDENTITY"}},
	{Model: "StandaloneMongodb", Field: "ssl_mode", Enum: []string{"allow", "prefer", "require", "verify-full"}},
}

func TestSchemaValidators_NotLooserThanContract(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(testdataDir(), "contracts", "coolify-v4.json"))
	if err != nil {
		t.Fatalf("reading pin contract: %v", err)
	}
	var c contractFile
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("parsing pin contract: %v", err)
	}

	schemaSrc, err := os.ReadFile(filepath.Join(testdataDir(), "..", "internal", "service", "application", "common_schema.go"))
	if err != nil {
		t.Fatalf("reading application schema: %v", err)
	}
	dbSrc, err := os.ReadFile(filepath.Join(testdataDir(), "..", "internal", "service", "database", "common.go"))
	if err != nil {
		t.Fatalf("reading database schema: %v", err)
	}

	for _, bound := range schemaValidatorBounds {
		model, ok := c.Models[bound.Model]
		if !ok {
			t.Errorf("contract missing model %s", bound.Model)
			continue
		}
		field, ok := model.Fields[bound.Field]
		if !ok {
			t.Errorf("contract %s missing field %s", bound.Model, bound.Field)
			continue
		}
		if bound.MaxLength > 0 && field.MaxLength != nil && bound.MaxLength > *field.MaxLength {
			t.Errorf("%s.%s schema LengthAtMost(%d) is looser than contract max_length %d",
				bound.Model, bound.Field, bound.MaxLength, *field.MaxLength)
		}
		if bound.Enum != nil && len(field.EnumValues) > 0 {
			for _, v := range bound.Enum {
				if !slices.Contains(field.EnumValues, v) {
					t.Errorf("%s.%s schema OneOf includes %q, not in contract enum_values %v",
						bound.Model, bound.Field, v, field.EnumValues)
				}
			}
		}
		src := schemaSrc
		marker := bound.Field
		if bound.Model != "Application" {
			src = dbSrc
			switch bound.Model {
			case "StandalonePostgresql":
				marker = "SSLModePostgresqlAttr"
			case "StandaloneMysql":
				marker = "SSLModeMysqlAttr"
			case "StandaloneMongodb":
				marker = "SSLModeMongodbAttr"
			}
		}
		block := schemaAttrBlock(src, marker)
		if len(block) == 0 {
			t.Errorf("schema source missing %s", marker)
			continue
		}
		if bound.MaxLength > 0 && !bytes.Contains(block, []byte(fmt.Sprintf("LengthAtMost(%d)", bound.MaxLength))) {
			t.Errorf("%s schema source missing LengthAtMost(%d)", bound.Field, bound.MaxLength)
		}
		for _, v := range bound.Enum {
			if !bytes.Contains(block, []byte(`"`+v+`"`)) {
				t.Errorf("%s schema source missing OneOf value %q", bound.Model, v)
			}
		}
		if bound.Model == "StandaloneMysql" && bytes.Contains(block, []byte(`"DISABLED"`)) {
			t.Error("MySQL ssl_mode schema still accepts DISABLED")
		}
		if bound.Model == "StandaloneMongodb" && bytes.Contains(block, []byte(`"verify-ca"`)) {
			t.Error("MongoDB ssl_mode schema still accepts verify-ca")
		}
	}
}

func schemaAttrBlock(src []byte, marker string) []byte {
	needles := [][]byte{
		[]byte("func " + marker),
		[]byte(`"` + marker + `":`),
	}
	var rest []byte
	for _, needle := range needles {
		if i := bytes.Index(src, needle); i >= 0 {
			rest = src[i:]
			break
		}
	}
	if rest == nil {
		return nil
	}
	if j := bytes.Index(rest[1:], []byte("\nfunc ")); j > 0 {
		return rest[:j+1]
	}
	if j := bytes.Index(rest, []byte("},\n\t\t\"")); j > 0 {
		return rest[:j]
	}
	if len(rest) > 1200 {
		return rest[:1200]
	}
	return rest
}

// uiOnlyOptionalDatabaseFields are Optional attributes that Coolify rejects on
// create and update. Flatten must not write GET defaults over a known plan (#880).
var uiOnlyOptionalDatabaseFields = []string{
	"ssl_mode",
	"enable_ssl",
	"is_log_drain_enabled",
	"is_include_timestamps",
}

func TestUIOnlyOptionalDatabaseFields_ListedForFlattenGuard(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(testdataDir(), "contracts", "coolify-v4.json"))
	if err != nil {
		t.Fatalf("reading pin contract: %v", err)
	}
	var c contractFile
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("parsing pin contract: %v", err)
	}

	create := map[string]struct{}{}
	update := map[string]struct{}{}
	for name, ep := range c.Endpoints {
		if name == "DatabasesController::create_database" || name == "DatabasesController::create" {
			for _, f := range ep.AllowedFields {
				create[f] = struct{}{}
			}
		}
		if name == "DatabasesController::update_by_uuid" || name == "DatabasesController::update" {
			for _, f := range ep.AllowedFields {
				update[f] = struct{}{}
			}
		}
	}
	if len(create) == 0 || len(update) == 0 {
		t.Fatalf("database create/update allow lists not found (create=%d update=%d)", len(create), len(update))
	}
	for _, field := range uiOnlyOptionalDatabaseFields {
		_, onCreate := create[field]
		_, onUpdate := update[field]
		if (onCreate || onUpdate) && len(create) > 0 {
			t.Errorf("field %s is on a write allow-list; remove it from uiOnlyOptionalDatabaseFields", field)
		}
	}
}

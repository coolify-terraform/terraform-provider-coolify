package validate

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// uuidRegexp matches RFC 4122 UUIDs (e.g. "550e8400-e29b-41d4-a716-446655440000")
// and Coolify NanoID-style identifiers (e.g. "deey8xhb2bm3fxpobcxyddfv").
var uuidRegexp = regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[0-9a-zA-Z]{20,36})$`)

// UUID returns a string validator that checks if the value is a valid UUID
// or Coolify identifier format.
func UUID() validator.String {
	return stringvalidator.RegexMatches(uuidRegexp, "must be a valid UUID (e.g. \"550e8400-e29b-41d4-a716-446655440000\") or Coolify identifier")
}

// IsUUID returns true if the string is a valid UUID or Coolify identifier format.
func IsUUID(s string) bool {
	return uuidRegexp.MatchString(s)
}

// ImportUUID validates that a string is a valid UUID or Coolify identifier
// for use in terraform import.
func ImportUUID(id string) error {
	if !IsUUID(id) {
		return fmt.Errorf("import ID %q is not a valid UUID or Coolify identifier", id)
	}
	return nil
}

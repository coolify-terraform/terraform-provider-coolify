package validate

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var uuidRegexp = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// UUID returns a string validator that checks if the value is a valid UUID format.
func UUID() validator.String {
	return stringvalidator.RegexMatches(uuidRegexp, "must be a valid UUID (e.g. \"550e8400-e29b-41d4-a716-446655440000\")")
}

// IsUUID returns true if the string is a valid UUID format.
func IsUUID(s string) bool {
	return uuidRegexp.MatchString(s)
}

// ImportUUID validates that a string is a valid UUID for use in terraform import.
// Returns a formatted error suitable for resp.Diagnostics.AddError if invalid.
func ImportUUID(id string) error {
	if !IsUUID(id) {
		return fmt.Errorf("import ID %q is not a valid UUID (expected format: 550e8400-e29b-41d4-a716-446655440000)", id)
	}
	return nil
}

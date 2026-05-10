package validate

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var uuidRegexp = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// UUID returns a string validator that checks if the value is a valid UUID format.
func UUID() validator.String {
	return stringvalidator.RegexMatches(uuidRegexp, "must be a valid UUID (e.g. \"550e8400-e29b-41d4-a716-446655440000\")")
}

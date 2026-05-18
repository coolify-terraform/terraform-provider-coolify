package validate

import (
	"fmt"
	"regexp"
	"strings"

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

// CompoundImportID holds the parsed components of a compound import ID
// in the format "project_uuid:server_uuid:environment_name:uuid".
type CompoundImportID struct {
	ProjectUUID     string
	ServerUUID      string
	EnvironmentName string
	UUID            string
}

// ParseCompoundImportID parses an import ID that is either a simple UUID
// or the compound format "project_uuid:server_uuid:environment_name:uuid".
// Returns the parsed parts and whether the compound format was used.
func ParseCompoundImportID(id string) (*CompoundImportID, bool, error) {
	parts := strings.SplitN(id, ":", 4)
	switch len(parts) {
	case 1:
		if err := ImportUUID(parts[0]); err != nil {
			return nil, false, err
		}
		return &CompoundImportID{UUID: parts[0]}, false, nil
	case 4:
		if err := ImportUUID(parts[0]); err != nil {
			return nil, false, fmt.Errorf("project_uuid: %w", err)
		}
		if err := ImportUUID(parts[1]); err != nil {
			return nil, false, fmt.Errorf("server_uuid: %w", err)
		}
		if parts[2] == "" {
			return nil, false, fmt.Errorf("environment_name must not be empty")
		}
		if err := ImportUUID(parts[3]); err != nil {
			return nil, false, fmt.Errorf("uuid: %w", err)
		}
		return &CompoundImportID{
			ProjectUUID:     parts[0],
			ServerUUID:      parts[1],
			EnvironmentName: parts[2],
			UUID:            parts[3],
		}, true, nil
	default:
		return nil, false, fmt.Errorf("expected UUID or project_uuid:server_uuid:environment_name:uuid, got: %s", id)
	}
}

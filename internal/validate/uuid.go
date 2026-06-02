package validate

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

// ImportParentChild handles the common ImportState pattern for resources
// that belong to a parent identified by type. It parses an import ID of
// the form "type:parent_uuid:child_uuid", validates all segments, and
// sets the state attributes (type_uuid for the parent, uuid for the child).
//
// allowedTypes lists valid type prefixes (e.g., "application", "service").
// childLabel is used in error messages (e.g., "storage", "task").
func ImportParentChild(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
	allowedTypes []string,
	childLabel string,
) {
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			parentChildFormatHint(allowedTypes, childLabel),
		)
		return
	}

	resourceType := parts[0]
	parentUUID := parts[1]
	childUUID := parts[2]

	if err := ImportUUID(parentUUID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("parent UUID segment: %s", err))
		return
	}
	if err := ImportUUID(childUUID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("%s UUID segment: %s", childLabel, err))
		return
	}

	valid := false
	for _, t := range allowedTypes {
		if resourceType == t {
			valid = true
			break
		}
	}
	if !valid {
		quoted := make([]string, len(allowedTypes))
		for i, t := range allowedTypes {
			quoted[i] = fmt.Sprintf("%q", t)
		}
		resp.Diagnostics.AddError(
			"Invalid import ID type",
			fmt.Sprintf("Expected %s, got %q.", joinOr(quoted), resourceType),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(resourceType+"_uuid"), parentUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), childUUID)...)
}

func parentChildFormatHint(allowedTypes []string, childLabel string) string {
	hints := make([]string, len(allowedTypes))
	for i, t := range allowedTypes {
		hints[i] = fmt.Sprintf("%q", t+":{parent_uuid}:{"+childLabel+"_uuid}")
	}
	return fmt.Sprintf("Expected %s.", joinOr(hints))
}

func joinOr(items []string) string {
	switch len(items) {
	case 1:
		return items[0]
	case 2:
		return items[0] + " or " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", or " + items[len(items)-1]
	}
}

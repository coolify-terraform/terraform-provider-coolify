package filter

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Block returns the schema block for the filter attribute.
func Block() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		MarkdownDescription: "Filter results by field values. Multiple filters are ANDed together. Multiple values within a filter are ORed.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "The field name to filter by.",
					Required:            true,
				},
				"values": schema.SetAttribute{
					MarkdownDescription: "The values to match against. Multiple values are ORed.",
					Required:            true,
					ElementType:         types.StringType,
				},
			},
		},
	}
}

// Config represents one filter block.
type Config struct {
	Name   types.String   `tfsdk:"name"`
	Values []types.String `tfsdk:"values"`
}

// Match checks if a field value matches any of the filter values.
// Supports exact match and glob patterns (* suffix).
func Match(value string, filterValues []types.String) bool {
	for _, fv := range filterValues {
		pattern := fv.ValueString()
		if pattern == value {
			return true
		}
		if strings.HasSuffix(pattern, "*") && strings.HasPrefix(value, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}
	return false
}

// Apply filters a slice of items using the filter configs and a field accessor function.
// The accessor returns the string value for a given field name, or ("", false) if the field doesn't exist.
func Apply[T any](items []T, filters []Config, accessor func(T, string) (string, bool)) []T {
	if len(filters) == 0 {
		return items
	}
	var result []T
	for _, item := range items {
		match := true
		for _, f := range filters {
			fieldName := f.Name.ValueString()
			val, ok := accessor(item, fieldName)
			if !ok {
				match = false
				break
			}
			if !Match(val, f.Values) {
				match = false
				break
			}
		}
		if match {
			result = append(result, item)
		}
	}
	return result
}

// BoolToString converts a bool to "true" or "false" for filter matching.
func BoolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// Int64ToString converts an int64 to a string for filter matching.
func Int64ToString(i int64) string {
	return fmt.Sprintf("%d", i)
}

// IntToString converts an int to a string for filter matching.
func IntToString(i int) string {
	return fmt.Sprintf("%d", i)
}

package notificationcommon

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// IDAttributeDS is the computed singleton id for notification data sources.
func IDAttributeDS() schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "Identifier. Always `current` (team is implied by the API token).",
		Computed:            true,
	}
}

// EnabledAttributeDS is the computed channel enable flag for data sources.
func EnabledAttributeDS(channel string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: fmt.Sprintf("Whether %s notifications are enabled for the team.", channel),
		Computed:            true,
	}
}

// BoolComputed is a computed bool attribute.
func BoolComputed(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: desc,
		Computed:            true,
	}
}

// EventDataSourceAttrs returns the 14 shared event bools as computed attributes.
func EventDataSourceAttrs(channel string) map[string]schema.Attribute {
	attrs := make(map[string]schema.Attribute, len(eventNames))
	for _, e := range eventNames {
		desc := fmt.Sprintf("Whether %s notifications are sent for %s events.", channel, e.label)
		attrs[e.attr] = BoolComputed(desc)
	}
	return attrs
}

// MergeDSAttrs returns a new map with base attributes plus overlay (overlay wins).
func MergeDSAttrs(base map[string]schema.Attribute, overlay map[string]schema.Attribute) map[string]schema.Attribute {
	out := make(map[string]schema.Attribute, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

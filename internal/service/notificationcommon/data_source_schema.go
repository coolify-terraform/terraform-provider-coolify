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

// EventDataSourceAttrs returns the 15 shared event bools as computed attributes.
func EventDataSourceAttrs(channel string) map[string]schema.Attribute {
	attrs := make(map[string]schema.Attribute, len(eventNames))
	for _, e := range eventNames {
		desc := fmt.Sprintf("Whether %s notifications are sent for %s events.", channel, e.label)
		if e.attr == "restart_limit_reached" {
			desc += " Coolify tip after 2026-08-31 (not in tag v4.3.14)."
		}
		attrs[e.attr] = BoolComputed(desc)
	}
	return attrs
}

// ThreadDataSourceAttrs returns the 15 Telegram forum thread IDs as computed attributes.
func ThreadDataSourceAttrs() map[string]schema.Attribute {
	attrs := make(map[string]schema.Attribute, len(eventNames))
	for _, e := range eventNames {
		desc := fmt.Sprintf("Forum thread ID for %s events. Coolify may omit this unless the API token can read sensitive fields (`read:sensitive` or root).", e.attr)
		attrs["thread_"+e.attr] = schema.StringAttribute{
			MarkdownDescription: desc,
			Computed:            true,
			Sensitive:           true,
		}
	}
	return attrs
}

// MergeDSAttrs returns a new map with later maps winning on key collision.
func MergeDSAttrs(maps ...map[string]schema.Attribute) map[string]schema.Attribute {
	n := 0
	for _, m := range maps {
		n += len(m)
	}
	out := make(map[string]schema.Attribute, n)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// Package notificationcommon holds shared schema and import helpers for
// team-scoped Coolify notification channel resources (Discord, Slack, email,
// Telegram, webhook, Pushover).
package notificationcommon

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// ImportIDCurrent is the only valid terraform import id for team-singleton
// notification resources. The team is implied by the API token.
const ImportIDCurrent = "current"

// eventNames is the ordered list of shared event bool attributes on Coolify
// notification channels (must stay aligned with Coolify channelConfig).
var eventNames = []struct {
	attr  string
	label string
}{
	{"deployment_success", "deployment success"},
	{"deployment_failure", "deployment failure"},
	{"status_change", "status change"},
	{"restart_limit_reached", "restart limit reached"},
	{"backup_success", "backup success"},
	{"backup_failure", "backup failure"},
	{"scheduled_task_success", "scheduled task success"},
	{"scheduled_task_failure", "scheduled task failure"},
	{"docker_cleanup_success", "Docker cleanup success"},
	{"docker_cleanup_failure", "Docker cleanup failure"},
	{"server_disk_usage", "server disk usage"},
	{"server_reachable", "server reachable"},
	{"server_unreachable", "server unreachable"},
	{"server_patch", "server patch"},
	{"traefik_outdated", "Traefik outdated"},
}

// EventAttributeNames returns the 15 shared event attribute names in stable order.
func EventAttributeNames() []string {
	out := make([]string, len(eventNames))
	for i, e := range eventNames {
		out[i] = e.attr
	}
	return out
}

// ThreadAttributeNames returns the 14 Telegram thread-id attribute names
// (thread_ + EventAttributeNames) in the same order.
func ThreadAttributeNames() []string {
	out := make([]string, len(eventNames))
	for i, e := range eventNames {
		out[i] = "thread_" + e.attr
	}
	return out
}

// StringOptComputedSensitive is Optional+Computed+Sensitive with UseStateForUnknown
// for Coolify fields that may be omitted on read without read:sensitive.
func StringOptComputedSensitive(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		Sensitive:           true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

// SensitiveOmitSuffix is appended to resource schema descriptions for values
// Coolify may hide without read:sensitive.
const SensitiveOmitSuffix = " Sensitive; Coolify may omit it on read unless the API token can read sensitive fields (`read:sensitive` or root). Preserve after import."

// StringSensitiveOmit is StringOptComputedSensitive with SensitiveOmitSuffix.
func StringSensitiveOmit(prefix string) schema.StringAttribute {
	return StringOptComputedSensitive(prefix + SensitiveOmitSuffix)
}

// BoolOptComputed is Optional+Computed with UseStateForUnknown for API-defaulted bools.
func BoolOptComputed(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

// EventSchemaAttrs returns the 15 shared event bool attributes for a channel.
// channel is the display name used in Markdown descriptions (e.g. "Discord", "email").
func EventSchemaAttrs(channel string) map[string]schema.Attribute {
	attrs := make(map[string]schema.Attribute, len(eventNames))
	for _, e := range eventNames {
		desc := fmt.Sprintf("Whether to send %s notifications for %s events.", channel, e.label)
		if e.attr == "restart_limit_reached" {
			desc += " Requires Coolify >= v4.3.15 (not in tag v4.3.14). Omit on older instances; the API rejects the field with HTTP 422."
		}
		attrs[e.attr] = BoolOptComputed(desc)
	}
	return attrs
}

// ThreadSchemaAttrs returns the 14 Telegram forum thread-id attributes.
// channel is the display name used in Markdown descriptions (e.g. "Telegram").
func ThreadSchemaAttrs(channel string) map[string]schema.Attribute {
	attrs := make(map[string]schema.Attribute, len(eventNames))
	for _, e := range eventNames {
		desc := fmt.Sprintf("%s forum thread ID for %s events. Sensitive; Coolify may omit it on read unless the API token can read sensitive fields (`read:sensitive` or root). Preserve after import.", channel, e.attr)
		attrs["thread_"+e.attr] = StringOptComputedSensitive(desc)
	}
	return attrs
}

// MergeAttrs returns a new map with later maps winning on key collision.
func MergeAttrs(maps ...map[string]schema.Attribute) map[string]schema.Attribute {
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

// IDAttribute is the computed singleton id attribute (always "current").
func IDAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "Resource identifier. Always `current` (team is implied by the API token).",
		Computed:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

// EnabledAttribute is the channel master enable flag (default false).
func EnabledAttribute(channel string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: fmt.Sprintf("Whether %s notifications are enabled for the team.", channel),
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
	}
}

// ImportIDError returns a non-nil error when id is not empty and not ImportIDCurrent.
// Empty id is accepted (terraform import sometimes passes blank then framework sets it).
func ImportIDError(typeName, id string) error {
	if id == "" || id == ImportIDCurrent {
		return nil
	}
	return fmt.Errorf("%s is a team singleton; import with id %q (got %q)", typeName, ImportIDCurrent, id)
}

// ImportStateCurrent validates the import id and sets state id to "current".
// typeName is the full resource type (e.g. coolify_notification_slack) for error text.
func ImportStateCurrent(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, typeName string) {
	if err := ImportIDError(typeName, req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ImportIDCurrent)...)
}

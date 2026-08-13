package notificationcommon_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventAttributeNames_CountAndOrder(t *testing.T) {
	t.Parallel()
	names := notificationcommon.EventAttributeNames()
	require.Len(t, names, 14)
	assert.Equal(t, "deployment_success", names[0])
	assert.Equal(t, "traefik_outdated", names[len(names)-1])
	seen := map[string]struct{}{}
	for _, n := range names {
		_, dup := seen[n]
		assert.False(t, dup, "duplicate %s", n)
		seen[n] = struct{}{}
	}
}

func TestEventSchemaAttrs_HasAllEventsAndChannelInDesc(t *testing.T) {
	t.Parallel()
	attrs := notificationcommon.EventSchemaAttrs("Discord")
	require.Len(t, attrs, 14)
	for _, name := range notificationcommon.EventAttributeNames() {
		a, ok := attrs[name]
		require.True(t, ok, "missing attr %s", name)
		ba, ok := a.(schema.BoolAttribute)
		require.True(t, ok, "%s should be BoolAttribute", name)
		assert.True(t, ba.Optional)
		assert.True(t, ba.Computed)
		assert.Contains(t, ba.MarkdownDescription, "Discord")
		assert.NotEmpty(t, ba.PlanModifiers)
	}
}

func TestMergeAttrs_OverlayWins(t *testing.T) {
	t.Parallel()
	base := map[string]schema.Attribute{
		"id":      notificationcommon.IDAttribute(),
		"enabled": notificationcommon.EnabledAttribute("Slack"),
	}
	overlay := notificationcommon.EventSchemaAttrs("Slack")
	merged := notificationcommon.MergeAttrs(base, overlay)
	require.Len(t, merged, 2+14)
	_, ok := merged["deployment_failure"]
	assert.True(t, ok)
	_, ok = merged["id"]
	assert.True(t, ok)
}

func TestImportIDError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, notificationcommon.ImportIDError("coolify_notification_slack", ""))
	assert.NoError(t, notificationcommon.ImportIDError("coolify_notification_slack", "current"))
	err := notificationcommon.ImportIDError("coolify_notification_slack", "not-current")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "team singleton")
	assert.Contains(t, err.Error(), "coolify_notification_slack")
	assert.Contains(t, err.Error(), "not-current")
}

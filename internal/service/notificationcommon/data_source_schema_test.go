package notificationcommon_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventDataSourceAttrs_AllEvents(t *testing.T) {
	t.Parallel()
	attrs := notificationcommon.EventDataSourceAttrs("Discord")
	require.Len(t, attrs, 15)
	for _, name := range notificationcommon.EventAttributeNames() {
		_, ok := attrs[name]
		assert.True(t, ok, "missing %s", name)
	}
}

func TestThreadDataSourceAttrs_AllThreads(t *testing.T) {
	t.Parallel()
	attrs := notificationcommon.ThreadDataSourceAttrs()
	require.Len(t, attrs, 15)
	for _, name := range notificationcommon.ThreadAttributeNames() {
		a, ok := attrs[name]
		require.True(t, ok, "missing %s", name)
		sa, ok := a.(schema.StringAttribute)
		require.True(t, ok, "%s should be StringAttribute", name)
		assert.True(t, sa.Computed)
		assert.True(t, sa.Sensitive)
	}
}

func TestMergeDSAttrs(t *testing.T) {
	t.Parallel()
	base := notificationcommon.EventDataSourceAttrs("Slack")
	merged := notificationcommon.MergeDSAttrs(base, map[string]schema.Attribute{
		"enabled": notificationcommon.EnabledAttributeDS("Slack"),
	})
	assert.Contains(t, merged, "enabled")
	assert.Contains(t, merged, "deployment_failure")
	_, ok := base["enabled"]
	assert.False(t, ok, "base must not be mutated")
}

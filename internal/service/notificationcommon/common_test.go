package notificationcommon_test

import (
	"context"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newImportStateResponse(s schema.Schema) *resource.ImportStateResponse {
	ctx := context.Background()
	return &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: s,
			Raw:    tftypes.NewValue(s.Type().TerraformType(ctx), nil),
		},
	}
}

func singletonSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": notificationcommon.IDAttribute(),
		},
	}
}

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
		// Colliding key: base uses email wording; overlay (EventSchemaAttrs) wins.
		"deployment_failure": notificationcommon.BoolOptComputed("base description must be replaced"),
	}
	overlay := notificationcommon.EventSchemaAttrs("Slack")
	merged := notificationcommon.MergeAttrs(base, overlay)
	require.Len(t, merged, 2+14)
	ba, ok := merged["deployment_failure"].(schema.BoolAttribute)
	require.True(t, ok)
	assert.Contains(t, ba.MarkdownDescription, "Slack")
	assert.NotContains(t, ba.MarkdownDescription, "base description")
	_, ok = merged["id"]
	assert.True(t, ok)
}

func TestIDAttribute_ComputedWithPlanModifiers(t *testing.T) {
	t.Parallel()
	a := notificationcommon.IDAttribute()
	assert.True(t, a.Computed)
	assert.False(t, a.Optional)
	assert.NotEmpty(t, a.PlanModifiers)
}

func TestEnabledAttribute_DefaultFalse(t *testing.T) {
	t.Parallel()
	a := notificationcommon.EnabledAttribute("Discord")
	assert.True(t, a.Optional)
	assert.True(t, a.Computed)
	assert.NotNil(t, a.Default)
	assert.Contains(t, a.MarkdownDescription, "Discord")
}

func TestBoolOptComputed_PlanModifiers(t *testing.T) {
	t.Parallel()
	a := notificationcommon.BoolOptComputed("test desc")
	assert.True(t, a.Optional)
	assert.True(t, a.Computed)
	assert.Equal(t, "test desc", a.MarkdownDescription)
	assert.NotEmpty(t, a.PlanModifiers)
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

func TestImportStateCurrent_SetsID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, id := range []string{"", "current"} {
		id := id
		t.Run("id="+id, func(t *testing.T) {
			t.Parallel()
			resp := newImportStateResponse(singletonSchema())
			notificationcommon.ImportStateCurrent(ctx, resource.ImportStateRequest{ID: id}, resp, "coolify_notification_test")
			require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics.Errors())
			var got types.String
			diags := resp.State.GetAttribute(ctx, path.Root("id"), &got)
			require.False(t, diags.HasError())
			assert.Equal(t, notificationcommon.ImportIDCurrent, got.ValueString())
		})
	}
}

func TestImportStateCurrent_RejectsOtherID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resp := newImportStateResponse(singletonSchema())
	notificationcommon.ImportStateCurrent(ctx, resource.ImportStateRequest{ID: "not-current"}, resp, "coolify_notification_slack")
	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Invalid import ID")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "team singleton")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "coolify_notification_slack")
}

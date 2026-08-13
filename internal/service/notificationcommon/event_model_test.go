package notificationcommon_test

import (
	"reflect"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventModel_AlignsWithEventAttributeNames(t *testing.T) {
	t.Parallel()
	names := notificationcommon.EventAttributeNames()
	require.Len(t, names, 14)

	typ := reflect.TypeOf(notificationcommon.EventModel{})
	require.Equal(t, len(names), typ.NumField())
	for i, name := range names {
		tag := typ.Field(i).Tag.Get("tfsdk")
		assert.Equal(t, name, tag, "EventModel field %d tfsdk", i)
	}
}

func TestEventModel_CreateUpdateAndFlatten(t *testing.T) {
	t.Parallel()
	plan := notificationcommon.EventModel{
		DeploymentFailure: types.BoolValue(true),
		ServerDiskUsage:   types.BoolValue(false),
	}
	ev := plan.CreateUpdate()
	require.NotNil(t, ev.DeploymentFailure)
	assert.True(t, *ev.DeploymentFailure)
	require.NotNil(t, ev.ServerDiskUsage)
	assert.False(t, *ev.ServerDiskUsage)
	assert.Nil(t, ev.DeploymentSuccess)

	var in client.UpdateDiscordNotificationInput
	require.NoError(t, client.ApplyEventUpdate(&in, ev))
	require.NotNil(t, in.DeploymentFailure)
	assert.True(t, *in.DeploymentFailure)

	src, err := client.EventsFrom(&client.DiscordNotificationSettings{
		DeploymentFailure: true,
		BackupFailure:     true,
	})
	require.NoError(t, err)
	var got notificationcommon.EventModel
	got.FlattenEvents(src)
	assert.True(t, got.DeploymentFailure.ValueBool())
	assert.True(t, got.BackupFailure.ValueBool())
	assert.False(t, got.DeploymentSuccess.ValueBool())
}

func TestEventModel_DiffUpdate(t *testing.T) {
	t.Parallel()
	plan := notificationcommon.EventModel{
		DeploymentFailure: types.BoolValue(true),
		BackupFailure:     types.BoolValue(false),
	}
	state := notificationcommon.EventModel{
		DeploymentFailure: types.BoolValue(true),
		BackupFailure:     types.BoolValue(true),
	}
	ev := plan.DiffUpdate(state)
	assert.Nil(t, ev.DeploymentFailure)
	require.NotNil(t, ev.BackupFailure)
	assert.False(t, *ev.BackupFailure)
}

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

func TestThreadModel_AlignsWithThreadAttributeNames(t *testing.T) {
	t.Parallel()
	names := notificationcommon.ThreadAttributeNames()
	require.Len(t, names, 14)

	typ := reflect.TypeOf(notificationcommon.ThreadModel{})
	require.Equal(t, len(names), typ.NumField())
	for i, name := range names {
		tag := typ.Field(i).Tag.Get("tfsdk")
		assert.Equal(t, name, tag, "ThreadModel field %d tfsdk", i)
	}
}

func TestThreadModel_CreateThreadUpdateAndFlatten(t *testing.T) {
	t.Parallel()
	plan := notificationcommon.ThreadModel{
		ThreadDeploymentFailure: types.StringValue("42"),
		ThreadServerDiskUsage:   types.StringValue(""),
	}
	th := plan.CreateThreadUpdate()
	require.NotNil(t, th.ThreadDeploymentFailure)
	assert.Equal(t, "42", *th.ThreadDeploymentFailure)
	require.NotNil(t, th.ThreadServerDiskUsage)
	assert.Empty(t, *th.ThreadServerDiskUsage)
	assert.Nil(t, th.ThreadDeploymentSuccess)

	var in client.UpdateTelegramNotificationInput
	require.NoError(t, client.ApplyThreadUpdate(&in, th))
	require.NotNil(t, in.ThreadDeploymentFailure)
	assert.Equal(t, "42", *in.ThreadDeploymentFailure)

	src, err := client.ThreadsFrom(&client.TelegramNotificationSettings{
		ThreadDeploymentFailure: "99",
		ThreadBackupFailure:     "7",
	})
	require.NoError(t, err)
	got := notificationcommon.ThreadModel{
		ThreadDeploymentSuccess: types.StringValue("keep-me"),
	}
	got.FlattenThreads(src)
	assert.Equal(t, "99", got.ThreadDeploymentFailure.ValueString())
	assert.Equal(t, "7", got.ThreadBackupFailure.ValueString())
	// API hid this sensitive field (empty); preserve prior state.
	assert.Equal(t, "keep-me", got.ThreadDeploymentSuccess.ValueString())
}

func TestThreadModel_DiffThreadUpdate(t *testing.T) {
	t.Parallel()
	plan := notificationcommon.ThreadModel{
		ThreadDeploymentFailure: types.StringValue("42"),
		ThreadBackupFailure:     types.StringValue("8"),
	}
	state := notificationcommon.ThreadModel{
		ThreadDeploymentFailure: types.StringValue("42"),
		ThreadBackupFailure:     types.StringValue("7"),
	}
	th := plan.DiffThreadUpdate(state)
	assert.Nil(t, th.ThreadDeploymentFailure)
	require.NotNil(t, th.ThreadBackupFailure)
	assert.Equal(t, "8", *th.ThreadBackupFailure)
}

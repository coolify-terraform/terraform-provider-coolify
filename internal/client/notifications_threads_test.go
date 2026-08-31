package client_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyThreadUpdate_Telegram(t *testing.T) {
	t.Parallel()
	fail := "42"
	th := client.NotificationThreadUpdate{
		ThreadDeploymentFailure: &fail,
	}
	var in client.UpdateTelegramNotificationInput
	require.NoError(t, client.ApplyThreadUpdate(&in, th))
	require.NotNil(t, in.ThreadDeploymentFailure)
	assert.Equal(t, "42", *in.ThreadDeploymentFailure)
	assert.Nil(t, in.ThreadDeploymentSuccess)
	assert.Nil(t, in.Token) // channel-specific left alone
}

func TestApplyThreadUpdate_AllFifteenFields(t *testing.T) {
	t.Parallel()
	vals := make([]string, 15)
	ptrs := make([]*string, 15)
	for i := range vals {
		vals[i] = string(rune('a' + i))
		ptrs[i] = &vals[i]
	}
	th := client.NotificationThreadUpdate{
		ThreadDeploymentSuccess:    ptrs[0],
		ThreadDeploymentFailure:    ptrs[1],
		ThreadStatusChange:         ptrs[2],
		ThreadRestartLimitReached:  ptrs[3],
		ThreadBackupSuccess:        ptrs[4],
		ThreadBackupFailure:        ptrs[5],
		ThreadScheduledTaskSuccess: ptrs[6],
		ThreadScheduledTaskFailure: ptrs[7],
		ThreadDockerCleanupSuccess: ptrs[8],
		ThreadDockerCleanupFailure: ptrs[9],
		ThreadServerDiskUsage:      ptrs[10],
		ThreadServerReachable:      ptrs[11],
		ThreadServerUnreachable:    ptrs[12],
		ThreadServerPatch:          ptrs[13],
		ThreadTraefikOutdated:      ptrs[14],
	}
	var in client.UpdateTelegramNotificationInput
	require.NoError(t, client.ApplyThreadUpdate(&in, th))
	got := []*string{
		in.ThreadDeploymentSuccess, in.ThreadDeploymentFailure, in.ThreadStatusChange,
		in.ThreadRestartLimitReached,
		in.ThreadBackupSuccess, in.ThreadBackupFailure,
		in.ThreadScheduledTaskSuccess, in.ThreadScheduledTaskFailure,
		in.ThreadDockerCleanupSuccess, in.ThreadDockerCleanupFailure,
		in.ThreadServerDiskUsage, in.ThreadServerReachable, in.ThreadServerUnreachable,
		in.ThreadServerPatch, in.ThreadTraefikOutdated,
	}
	require.Len(t, got, 15)
	for i, p := range got {
		require.NotNil(t, p, "field %d", i)
		assert.Equal(t, vals[i], *p, "field %d", i)
	}
	assert.Nil(t, in.Enabled)
	assert.Nil(t, in.Token)
	assert.Nil(t, in.ChatID)
}

func TestApplyThreadUpdate_RejectsUnknownDest(t *testing.T) {
	t.Parallel()
	var notTelegram struct{ Name string }
	err := client.ApplyThreadUpdate(&notTelegram, client.NotificationThreadUpdate{})
	require.Error(t, err)
}

func TestThreadsFrom_Telegram(t *testing.T) {
	t.Parallel()
	got, err := client.ThreadsFrom(&client.TelegramNotificationSettings{
		Enabled:                 true,
		Token:                   "secret",
		ThreadDeploymentFailure: "99",
		ThreadBackupFailure:     "7",
	})
	require.NoError(t, err)
	assert.Equal(t, "99", got.ThreadDeploymentFailure)
	assert.Equal(t, "7", got.ThreadBackupFailure)
	assert.Empty(t, got.ThreadDeploymentSuccess)
}

func TestThreadsFrom_Nil(t *testing.T) {
	t.Parallel()
	_, err := client.ThreadsFrom((*client.TelegramNotificationSettings)(nil))
	require.Error(t, err)
}

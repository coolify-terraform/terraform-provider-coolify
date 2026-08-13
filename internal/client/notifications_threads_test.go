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

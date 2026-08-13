package client_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyEventUpdate_AllChannels(t *testing.T) {
	t.Parallel()
	yes := true
	ev := client.NotificationEventUpdate{
		DeploymentFailure: &yes,
		ServerDiskUsage:   &yes,
	}

	t.Run("discord", func(t *testing.T) {
		t.Parallel()
		var in client.UpdateDiscordNotificationInput
		require.NoError(t, client.ApplyEventUpdate(&in, ev))
		require.NotNil(t, in.DeploymentFailure)
		assert.True(t, *in.DeploymentFailure)
		require.NotNil(t, in.ServerDiskUsage)
		assert.True(t, *in.ServerDiskUsage)
		assert.Nil(t, in.DeploymentSuccess)
		assert.Nil(t, in.Enabled) // channel-specific left alone
	})
	t.Run("slack", func(t *testing.T) {
		t.Parallel()
		var in client.UpdateSlackNotificationInput
		require.NoError(t, client.ApplyEventUpdate(&in, ev))
		require.NotNil(t, in.DeploymentFailure)
		assert.True(t, *in.DeploymentFailure)
	})
	t.Run("webhook", func(t *testing.T) {
		t.Parallel()
		var in client.UpdateWebhookNotificationInput
		require.NoError(t, client.ApplyEventUpdate(&in, ev))
		require.NotNil(t, in.ServerDiskUsage)
		assert.True(t, *in.ServerDiskUsage)
	})
	t.Run("pushover", func(t *testing.T) {
		t.Parallel()
		var in client.UpdatePushoverNotificationInput
		require.NoError(t, client.ApplyEventUpdate(&in, ev))
		require.NotNil(t, in.DeploymentFailure)
		assert.True(t, *in.DeploymentFailure)
	})
	t.Run("email", func(t *testing.T) {
		t.Parallel()
		var in client.UpdateEmailNotificationInput
		require.NoError(t, client.ApplyEventUpdate(&in, ev))
		require.NotNil(t, in.DeploymentFailure)
		assert.True(t, *in.DeploymentFailure)
	})
	t.Run("telegram", func(t *testing.T) {
		t.Parallel()
		var in client.UpdateTelegramNotificationInput
		require.NoError(t, client.ApplyEventUpdate(&in, ev))
		require.NotNil(t, in.DeploymentFailure)
		assert.True(t, *in.DeploymentFailure)
	})
}

func TestEventsFrom_AllChannels(t *testing.T) {
	t.Parallel()
	t.Run("discord", func(t *testing.T) {
		t.Parallel()
		got, err := client.EventsFrom(&client.DiscordNotificationSettings{
			Enabled:           true,
			DeploymentFailure: true,
			ServerDiskUsage:   true,
		})
		require.NoError(t, err)
		assert.True(t, got.DeploymentFailure)
		assert.True(t, got.ServerDiskUsage)
		assert.False(t, got.DeploymentSuccess)
	})
	t.Run("slack", func(t *testing.T) {
		t.Parallel()
		got, err := client.EventsFrom(&client.SlackNotificationSettings{BackupFailure: true})
		require.NoError(t, err)
		assert.True(t, got.BackupFailure)
	})
	t.Run("webhook", func(t *testing.T) {
		t.Parallel()
		got, err := client.EventsFrom(&client.WebhookNotificationSettings{StatusChange: true})
		require.NoError(t, err)
		assert.True(t, got.StatusChange)
	})
	t.Run("pushover", func(t *testing.T) {
		t.Parallel()
		got, err := client.EventsFrom(&client.PushoverNotificationSettings{TraefikOutdated: true})
		require.NoError(t, err)
		assert.True(t, got.TraefikOutdated)
	})
	t.Run("email", func(t *testing.T) {
		t.Parallel()
		got, err := client.EventsFrom(&client.EmailNotificationSettings{ServerPatch: true})
		require.NoError(t, err)
		assert.True(t, got.ServerPatch)
	})
	t.Run("telegram", func(t *testing.T) {
		t.Parallel()
		got, err := client.EventsFrom(&client.TelegramNotificationSettings{DockerCleanupFailure: true})
		require.NoError(t, err)
		assert.True(t, got.DockerCleanupFailure)
	})
}

func TestApplyEventUpdate_RejectsUnknownDest(t *testing.T) {
	t.Parallel()
	var notAChannel struct{ Name string }
	err := client.ApplyEventUpdate(&notAChannel, client.NotificationEventUpdate{})
	require.Error(t, err)
}

func TestEventsFrom_Nil(t *testing.T) {
	t.Parallel()
	_, err := client.EventsFrom((*client.DiscordNotificationSettings)(nil))
	require.Error(t, err)
}
